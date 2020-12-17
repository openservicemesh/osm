package configurator

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
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

	// boolFieldsInConfigMap are the fields in osm-config that take in a boolean
	boolFieldsInConfigMap = []string{"egress", "enable_debug_server", "permissive_traffic_policy_mode", "prometheus_scraping", "tracing_enable", "use_https_ingress"}

	// ValidEnvoyLogLevels is a list of envoy log levels
	ValidEnvoyLogLevels = []string{"trace", "debug", "info", "warning", "warn", "error", "critical", "off"}
)

const (
	// ValidatingWebhookName is the name of the validating webhook used for validating osm-config
	ValidatingWebhookName = "validating-webhook.k8s.io"

	// webhookUpdateConfigMapis the HTTP path at which the webhook expects to receive configmap update events
	webhookUpdateConfigMap = "/validate-webhook"
	listenPort             = 9090

	// MustBeBool is the reason for denial for a boolean field
	MustBeBool = ": must be a boolean"

	// MustBeValidLogLvl is the reason for denial for envoy_log_level field
	MustBeValidLogLvl = ": invalid log level"

	// MustBeValidTime is the reason for denial for incorrect syntax for service_cert_validity_duration field
	MustBeValidTime = ": invalid time format must be a sequence of decimal numbers each with optional fraction and a unit suffix"

	// MustBeLessThanAYear is the reason for denial for service_cert_validity_duration field
	MustBeLessThanAYear = ": must be max 8760H (1 year)"

	// MustFollowSyntax is the reason for denial for tracing_address field
	MustFollowSyntax = ": invalid hostname syntax"

	// MustbeInt is the reason for denial for incorrect syntax for tracing_port field
	MustbeInt = ": must be an integer"

	// MustBeInPortRange is the reason for denial for tracing_port field
	MustBeInPortRange = ": must be between 0 and 65535"

	// CannotChangeMetadata is the reason for denial for changes to configmap metadata
	CannotChangeMetadata = ": cannot change metadata"

	hrInAYear  = 8760
	maxPortNum = 65535
)

type webhookConfig struct {
	kubeClient   kubernetes.Interface
	cert         certificate.Certificater
	certManager  certificate.Manager
	osmNamespace string
}

// NewWebhookConfig  starts a new web server handling requests from the  ValidatingWebhookConfiguration
func NewWebhookConfig(kubeClient kubernetes.Interface, certManager certificate.Manager, osmNamespace, webhookConfigName string, stop <-chan struct{}) error {
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMControllerName, osmNamespace))
	cert, err := certManager.IssueCertificate(cn, constants.XDSCertificateValidityPeriod)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing certificate for the validating webhook: %+v", err)
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
		log.Error().Err(err).Msgf("Error configuring ValidatingWebhookConfiguration %s: %+v", webhookConfigName, err)
		return err
	}
	return nil
}

func (whc *webhookConfig) runValidatingWebhook(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.DefaultServeMux

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
			log.Error().Err(err).Msgf("Error parsing webhook certificate: %+v", err)
			return
		}

		// #nosec G402
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Error().Err(err).Msgf("Validating webhook HTTP server failed to start: %+v", err)
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

	whc.validate(configMap, resp)

	return resp
}

// validate performs all the checks to the configmap fields and rejects as necessary
func (whc *webhookConfig) validate(configMap corev1.ConfigMap, resp *v1beta1.AdmissionResponse) *v1beta1.AdmissionResponse {
	for field, value := range configMap.Data {
		if !checkBoolFields(field, value, boolFieldsInConfigMap) {
			reasonForDenial(resp, MustBeBool, field)
		}
		if field == "envoy_log_level" && !checkEnvoyLogLevels(field, value, ValidEnvoyLogLevels) {
			reasonForDenial(resp, MustBeValidLogLvl, field)
		}
		if field == "service_cert_validity_duration" {
			t, err := time.ParseDuration(value)
			if err != nil {
				reasonForDenial(resp, MustBeValidTime, field)
			}
			if t.Hours() > hrInAYear {
				reasonForDenial(resp, MustBeLessThanAYear, field)
			}
		}
		if field == "tracing_address" && !matchAddrSyntax(field, value) {
			reasonForDenial(resp, MustFollowSyntax, field)
		}
		if field == "tracing_port" {
			portNum, err := strconv.Atoi(value)
			if err != nil {
				reasonForDenial(resp, MustbeInt, field)
			}
			if portNum < 0 || portNum > maxPortNum {
				reasonForDenial(resp, MustBeInPortRange, field)
			}
		}
	}

	originalConfigMap, _ := whc.kubeClient.CoreV1().ConfigMaps(whc.osmNamespace).Get(context.TODO(), constants.OSMConfigMap, metav1.GetOptions{})

	for metadataAnnotation, val := range configMap.ObjectMeta.Annotations {
		if originalConfigMap.Annotations[metadataAnnotation] != val {
			reasonForDenial(resp, CannotChangeMetadata, metadataAnnotation)
		}
	}
	for metadataLabels, val := range configMap.ObjectMeta.Labels {
		if originalConfigMap.Labels[metadataLabels] != val {
			reasonForDenial(resp, CannotChangeMetadata, metadataLabels)
		}
	}
	return resp
}

// matchAddrSyntax checks that string follows hostname syntax
func matchAddrSyntax(configMapField, configMapValue string) bool {
	syntax := regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)

	return syntax.Match([]byte(configMapValue))
}

// checkEnvoyLogLevels checks that the field value is a valid log level
func checkEnvoyLogLevels(configMapField, configMapValue string, levels []string) bool {
	valid := false
	for _, lvl := range levels {
		if configMapValue == lvl {
			valid = true
		}
	}
	return valid
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

func reasonForDenial(resp *v1beta1.AdmissionResponse, mustBe string, field string) {
	reason := "\n" + field + mustBe
	resp.Result = &metav1.Status{
		Reason: resp.Result.Reason + metav1.StatusReason(reason),
	}
	resp.Allowed = false
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
