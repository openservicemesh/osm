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

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/webhook"
	"github.com/pkg/errors"
	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

var (
	codecs       = serializer.NewCodecFactory(runtime.NewScheme())
	deserializer = codecs.UniversalDeserializer()

	//boolFieldsInConfigMap are the fields in osm-config that take in a boolean
	boolFieldsInConfigMap = []string{"egress", "enable_debug_server", "permissive_traffic_policy_mode", "prometheus_scraping", "tracing_enable", "use_https_ingress"}

	//ValidEnvoyLogLevels is a list of envoy log levels
	ValidEnvoyLogLevels = []string{"trace", "debug", "info", "warning", "warn", "error", "critical", "off"}
)

const (
	// ValidatingWebhookName is the name of the validating webhook used for validating osm-config
	ValidatingWebhookName = "validating-webhook.k8s.io"

	mustBeBool           = ": must be a boolean"
	mustBeValidLogLvl    = ": invalid log level"
	mustBeValidTime      = ": invalid time format must be a sequence of decimal numbers each with optional fraction and a unit suffix"
	mustBeLessThanAYear  = ": must be max 8760H (1 year)"
	mustFollowSyntax     = ": invalid hostname syntax"
	mustbeInt            = ": must be an integer"
	mustBeInPortRange    = ": must be between 0 and 65535"
	cannotChangeMetadata = ": cannot change metadata"
	hrInAYear            = 8760
	maxPortNum           = 65535
)

type webhookConfig struct {
	kubeClient   kubernetes.Interface
	cert         certificate.Certificater
	certManager  certificate.Manager
	osmNamespace string
}

//NewWebhookConfig thing stuff
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
	go whc.runValidatingWebhook(stop)

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

	mux.HandleFunc("/validate-webhook", whc.configMapHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", 9090),
		Handler: mux,
	}
	// start webhook server in new rountine
	go func() {
		if whc.cert == nil {
			log.Error().Msgf("Error certificate is nil")
			return
		}
		cert, err := tls.X509KeyPair(whc.cert.GetCertificateChain(), whc.cert.GetPrivateKey())
		if err != nil {
			log.Error().Err(err).Msgf("Error parsing webhook certificate: %+v", err)
			return
		}
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
	admissionRequestBody, err := webhook.GetAdmissionRequestBody(w, req)

	requestForNamespace, admissionResp := whc.getAdmissionReqResp(admissionRequestBody)

	resp, err := json.Marshal(&admissionResp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshalling admission response: %s", err), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(resp); err != nil {
		log.Error().Err(err).Msgf("Error writing admission response for pod in namespace %s", requestForNamespace)
	}
}

func (whc *webhookConfig) getAdmissionReqResp(admissionRequestBody []byte) (requestForNamespace string, admissionResp v1beta1.AdmissionReview) {
	var admissionReq v1beta1.AdmissionReview
	if _, _, err := deserializer.Decode(admissionRequestBody, nil, &admissionReq); err != nil {
		admissionResp.Response = webhook.AdmissionError(err)
	} else {
		admissionResp.Response = whc.validateConfigMap(admissionReq.Request)
	}

	return admissionReq.Request.Namespace, admissionResp
}

func (whc *webhookConfig) validateConfigMap(req *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	// fmt.Println("HI", req)
	// fmt.Println("HI marshal", string(req.Object.Raw))
	if req == nil {
		return webhook.AdmissionError(errors.New("nil Admission Request"))
	}

	// Decode the configmap from the request
	var configMap corev1.ConfigMap
	if _, _, err := deserializer.Decode(req.Object.Raw, nil, &configMap); err != nil {
		fmt.Println(&configMap)
		fmt.Println(string(req.Object.Raw))
		return webhook.AdmissionError(err)
	}

	//default response
	resp := &v1beta1.AdmissionResponse{
		Allowed: true,
		Result:  &metav1.Status{Reason: ""},
		UID:     req.UID,
	}

	//Allow if not a change to osm-config
	if req.Name != constants.OSMConfigMap {
		return resp
	}

	whc.validate(configMap, resp)

	return resp
}

//validate performs all the checks to the configmap fields and rejects as necessary
func (whc *webhookConfig) validate(configMap corev1.ConfigMap, resp *v1beta1.AdmissionResponse) *v1beta1.AdmissionResponse {
	for field, value := range configMap.Data {
		if !checkBoolFields(field, value, boolFieldsInConfigMap) {
			reasonForDenial(resp, mustBeBool, field)
		}
		if field == "envoy_log_level" && !checkEnvoyLogLevels(field, value, ValidEnvoyLogLevels) {
			reasonForDenial(resp, mustBeValidLogLvl, field)
		}
		if field == "service_cert_validity_duration" {
			t, err := time.ParseDuration(value)
			if err != nil {
				reasonForDenial(resp, mustBeValidTime, field)
			}
			if t.Hours() > hrInAYear {
				reasonForDenial(resp, mustBeLessThanAYear, field)
			}
		}
		if field == "tracing_address" && !matchAddrSyntax(field, value) {
			reasonForDenial(resp, mustFollowSyntax, field)
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
	}

	originalConfigMap, _ := whc.kubeClient.CoreV1().ConfigMaps(whc.osmNamespace).Get(context.TODO(), constants.OSMConfigMap, metav1.GetOptions{})

	for metadataAnnotation, val := range configMap.ObjectMeta.Annotations {
		if originalConfigMap.Annotations[metadataAnnotation] != val {
			reasonForDenial(resp, cannotChangeMetadata, metadataAnnotation)
		}
	}
	for metadataLabels, val := range configMap.ObjectMeta.Labels {
		if originalConfigMap.Labels[metadataLabels] != val {
			reasonForDenial(resp, cannotChangeMetadata, metadataLabels)
		}
	}
	return resp
}

//matchAddrSyntax checks that string follows hostname syntax
func matchAddrSyntax(configMapField, configMapValue string) bool {
	syntax := regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	if syntax.Match([]byte(configMapValue)) {
		return true
	}
	return false
}

//checkEnvoyLogLevels checks that the field value is a valid log level
func checkEnvoyLogLevels(configMapField, configMapValue string, levels []string) bool {
	valid := false
	for _, lvl := range levels {
		if configMapValue == lvl {
			valid = true
		}
	}
	if !valid {
		return false
	}
	return true
}

//checkBoolFields checks that the value is a boolean for fields that take in a boolean
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
