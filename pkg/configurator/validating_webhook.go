package configurator

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/webhook"
)

var (
	codecs       = serializer.NewCodecFactory(runtime.NewScheme())
	deserializer = codecs.UniversalDeserializer()

	// boolFields are the fields in osm-config that take in a boolean
	boolFields = []string{"egress", "enable_debug_server", "permissive_traffic_policy_mode", "prometheus_scraping", "tracing_enable", "use_https_ingress"}

	// ValidEnvoyLogLevels is a list of envoy log levels
	ValidEnvoyLogLevels = []string{"trace", "debug", "info", "warning", "warn", "error", "critical", "off"}

	// defaultFields are the default fields in osm-config
	defaultFields = []string{"egress", "enable_debug_server", "permissive_traffic_policy_mode", "prometheus_scraping", "tracing_enable", "use_https_ingress", "envoy_log_level", "service_cert_validity_duration", "tracing_address", "tracing_port", "tracing_endpoint"}
)

const (
	// ValidatingWebhookName is the name of the validating webhook used for validating osm-config
	ValidatingWebhookName = "osm-config-webhook.k8s.io"

	// webhookUpdateConfigMapis the HTTP path at which the webhook expects to receive configmap update events
	webhookUpdateConfigMap = "/validate-webhook"

	// listenPort is the validating webhook server port
	listenPort = 9093

	// mustBeBool is the reason for denial for a boolean field
	mustBeBool = ": must be a boolean"

	// mustBeValidLogLvl is the reason for denial for envoy_log_level field
	mustBeValidLogLvl = ": invalid log level"

	// mustBeValidTime is the reason for denial for incorrect syntax for service_cert_validity_duration field
	mustBeValidTime = ": invalid time format must be a sequence of decimal numbers each with optional fraction and a unit suffix"

	// mustbeInt is the reason for denial for incorrect syntax for tracing_port field
	mustbeInt = ": must be an integer"

	// mustBeInPortRange is the reason for denial for tracing_port field
	mustBeInPortRange = ": must be between 0 and 65535"

	mustBeValidIPRange = ": must be a list of valid IP addresses of the form a.b.c.d/x"

	// cannotChangeMetadata is the reason for denial for changes to configmap metadata
	cannotChangeMetadata = ": cannot change metadata"

	// doesNotContainDef is the reason for denial for not having default field(s) for osm-config
	doesNotContainDef = ": must be included as it is a default field"

	maxPortNum = 65535

	validatorServiceName = "osm-config-validator"
)

type webhookConfig struct {
	kubeClient   kubernetes.Interface
	cert         certificate.Certificater
	certManager  certificate.Manager
	osmNamespace string
}

// NewValidatingWebhook  starts a new web server handling requests from the  ValidatingWebhookConfiguration
func NewValidatingWebhook(kubeClient kubernetes.Interface, certManager certificate.Manager, osmNamespace, webhookConfigName string, stop <-chan struct{}) error {
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.svc", validatorServiceName, osmNamespace))
	cert, err := certManager.IssueCertificate(cn, constants.XDSCertificateValidityPeriod)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing certificate for the validating webhook")
		return err
	}

	whc := &webhookConfig{
		kubeClient:   kubeClient,
		certManager:  certManager,
		osmNamespace: osmNamespace,
		cert:         cert,
	}

	// Start the ValidatingWebhook web server
	go whc.runValidatingWebhook(stop)

	// Update the ValidatingWebhookConfig with the OSM CA bundle
	if err = updateValidatingWebhookCABundle(cert, webhookConfigName, whc.kubeClient); err != nil {
		log.Error().Err(err).Msgf("Error configuring ValidatingWebhookConfiguration %s", webhookConfigName)
		return err
	}
	return nil
}

func (whc *webhookConfig) runValidatingWebhook(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()

	mux.HandleFunc(webhookUpdateConfigMap, whc.configMapHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", listenPort),
		Handler: mux,
	}

	log.Info().Msgf("Starting configmap webhook server on port: %v", listenPort)

	go func() {
		if whc.cert == nil {
			log.Error().Msgf("Error certificate is nil")
			return
		}

		// Generate a key pair from your pem-encoded cert and key ([]byte).
		cert, err := tls.X509KeyPair(whc.cert.GetCertificateChain(), whc.cert.GetPrivateKey())
		if err != nil {
			log.Error().Err(err).Msg("Error parsing webhook certificate")
			return
		}

		// #nosec G402
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Error().Err(err).Msg("Validating webhook HTTP server failed to start")
			return
		}
	}()

	// Wait on exit signals
	<-stop
	// Stop the server
	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error shutting down validating webhook HTTP server")
	} else {
		log.Info().Msg("Done shutting down validating webhook HTTP server")
	}
}
func (whc *webhookConfig) configMapHandler(w http.ResponseWriter, req *http.Request) {
	log.Trace().Msgf("Received validating webhook request: Method=%v, URL=%v", req.Method, req.URL)

	admissionRequestBody, err := webhook.GetAdmissionRequestBody(w, req)
	if err != nil {
		// Error was already logged and written to the ResponseWriter
		return
	}

	requestForNamespace, admissionResp := whc.getAdmissionReqResp(admissionRequestBody)

	resp, err := json.Marshal(&admissionResp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshalling admission response: %s", err), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Error marshalling admission response; Responded to admission request for configmap in namespace %s with HTTP %v", requestForNamespace, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(resp); err != nil {
		log.Error().Err(err).Msgf("Error writing admission response for pod in namespace %s", requestForNamespace)
	}
}

func (whc *webhookConfig) getAdmissionReqResp(admissionRequestBody []byte) (requestForNamespace string, admissionResp v1beta1.AdmissionReview) {
	var admissionReq v1beta1.AdmissionReview
	if _, _, err := deserializer.Decode(admissionRequestBody, nil, &admissionReq); err != nil {
		log.Error().Err(err).Msg("Error decoding admission request body")
		admissionResp.Response = webhook.AdmissionError(err)
	} else {
		admissionResp.Response = whc.validateConfigMap(admissionReq.Request)
	}

	return admissionReq.Request.Namespace, admissionResp
}

func (whc *webhookConfig) validateConfigMap(req *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	if req == nil {
		log.Error().Msg("nil admission request")
		return webhook.AdmissionError(errNilAdmissionRequest)
	}

	// Decode the configmap from the request
	var configMap corev1.ConfigMap
	if _, _, err := deserializer.Decode(req.Object.Raw, nil, &configMap); err != nil {
		log.Error().Err(err).Msgf("Error unmarshaling request to configmap in namespace %s", req.Namespace)
		return webhook.AdmissionError(err)
	}

	log.Trace().Msgf("Validation request: (new object: %v) (old object: %v)", string(req.Object.Raw), string(req.OldObject.Raw))

	// Default response
	resp := &v1beta1.AdmissionResponse{
		Allowed: true,
		Result:  &metav1.Status{Reason: ""},
		UID:     req.UID,
	}

	// Allow if not a change to osm-config
	if req.Name != constants.OSMConfigMap {
		return resp
	}

	checkDefaultFields(configMap, resp)
	whc.validateFields(configMap, resp)

	return resp
}

// checkDefaultFields checks that all default fields for osm-config exist
func checkDefaultFields(configMap corev1.ConfigMap, resp *v1beta1.AdmissionResponse) *v1beta1.AdmissionResponse {
	data := make(map[string]struct{})
	for field := range configMap.Data {
		data[field] = struct{}{}
	}

	for _, f := range defaultFields {
		if _, ok := data[f]; !ok {
			reasonForDenial(resp, doesNotContainDef, f)
		}
	}
	return resp
}

// validateFields checks whether the configmap field values are valid and rejects as necessary
func (whc *webhookConfig) validateFields(configMap corev1.ConfigMap, resp *v1beta1.AdmissionResponse) *v1beta1.AdmissionResponse {
	for field, value := range configMap.Data {
		if !checkBoolFields(field, value, boolFields) {
			reasonForDenial(resp, mustBeBool, field)
		}
		if field == "envoy_log_level" && !checkEnvoyLogLevels(field, value) {
			reasonForDenial(resp, mustBeValidLogLvl, field)
		}
		if field == "service_cert_validity_duration" {
			_, err := time.ParseDuration(value)
			if err != nil {
				reasonForDenial(resp, mustBeValidTime, field)
			}
		}
		if field == "tracing_port" {
			portNum, err := strconv.Atoi(value)
			if err != nil {
				reasonForDenial(resp, mustbeInt, field)
			}
			if portNum < 0 || portNum > maxPortNum {
				reasonForDenial(resp, mustBeInPortRange, field)
			}
		}
		if field == outboundIPRangeExclusionListKey && !checkOutboundIPRangeExclusionList(value) {
			reasonForDenial(resp, mustBeValidIPRange, field)
		}
	}

	defConfigMap, _ := whc.kubeClient.CoreV1().ConfigMaps(whc.osmNamespace).Get(context.TODO(), constants.OSMConfigMap, metav1.GetOptions{})

	for metadataAnnotation, val := range configMap.ObjectMeta.Annotations {
		if defConfigMap.Annotations[metadataAnnotation] != val {
			reasonForDenial(resp, cannotChangeMetadata, metadataAnnotation)
		}
	}
	for metadataLabels, val := range configMap.ObjectMeta.Labels {
		if defConfigMap.Labels[metadataLabels] != val {
			reasonForDenial(resp, cannotChangeMetadata, metadataLabels)
		}
	}
	return resp
}

// checkEnvoyLogLevels checks that the field value is a valid log level
func checkEnvoyLogLevels(configMapField, configMapValue string) bool {
	valid := false
	for _, lvl := range ValidEnvoyLogLevels {
		if configMapValue == lvl {
			valid = true
		}
	}
	return valid
}

func checkOutboundIPRangeExclusionList(ipRangesStr string) bool {
	exclusionList := strings.Split(ipRangesStr, ",")
	for i := range exclusionList {
		ipAddress := strings.TrimSpace(exclusionList[i])
		if _, _, err := net.ParseCIDR(ipAddress); err != nil {
			return false
		}
	}
	return true
}

// checkBoolFields checks that the value is a boolean for fields that take in a boolean
func checkBoolFields(configMapField, configMapValue string, fields []string) bool {
	for _, f := range fields {
		if configMapField == f {
			_, err := strconv.ParseBool(configMapValue)
			if err != nil {
				return false
			}
		}
	}
	return true
}

// reasonForDenial rejects and appends rejection reason(s) to v1beta1.AdmissionResponse
func reasonForDenial(resp *v1beta1.AdmissionResponse, mustBe string, field string) {
	resp.Allowed = false
	reason := "\n" + field + mustBe
	resp.Result = &metav1.Status{
		Reason: resp.Result.Reason + metav1.StatusReason(reason),
	}
}

// getPartialValidatingWebhookConfiguration returns only the portion of the ValidatingWebhookConfiguration that needs to be updated.
func getPartialValidatingWebhookConfiguration(webhookName string, cert certificate.Certificater, webhookConfigName string) admissionv1beta1.ValidatingWebhookConfiguration {
	return admissionv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookConfigName,
		},
		Webhooks: []admissionv1beta1.ValidatingWebhook{
			{
				Name: webhookName,
				ClientConfig: admissionv1beta1.WebhookClientConfig{
					CABundle: cert.GetCertificateChain(),
				},
			},
		},
	}
}

// updateValidatingWebhookCABundle updates the existing ValidatingWebhookConfiguration with the CA this OSM instance runs with.
// It is necessary to perform this patch because the original ValidatingWebhookConfig YAML does not contain the root certificate.
func updateValidatingWebhookCABundle(cert certificate.Certificater, webhookName string, clientSet kubernetes.Interface) error {
	vwc := clientSet.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations()
	if _, err := vwc.Get(context.Background(), webhookName, metav1.GetOptions{}); err != nil {
		log.Error().Err(err).Msgf("Error getting ValidatingWebhookConfiguration %s; Will not update CA Bundle for webhook", webhookName)
		return err
	}

	patchJSON, err := json.Marshal(getPartialValidatingWebhookConfiguration(ValidatingWebhookName, cert, webhookName))
	if err != nil {
		return err
	}

	if _, err = vwc.Patch(context.Background(), webhookName, types.StrategicMergePatchType, patchJSON, metav1.PatchOptions{}); err != nil {
		log.Error().Err(err).Msgf("Error updating CA Bundle for ValidatingWebhookConfiguration %s", webhookName)
		return err
	}

	log.Info().Msgf("Finished updating CA Bundle for ValidatingWebhookConfiguration %s", webhookName)
	return nil
}
