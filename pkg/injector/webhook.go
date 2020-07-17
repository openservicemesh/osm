package injector

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/namespace"
)

var (
	codecs       = serializer.NewCodecFactory(runtime.NewScheme())
	deserializer = codecs.UniversalDeserializer()

	kubeSystemNamespaces = []string{
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
	}
)

const (
	osmWebhookName       = "osm-inject.k8s.io"
	osmWebhookMutatePath = "/mutate"
)

// NewWebhook starts a new web server handling requests from the injector MutatingWebhookConfiguration
func NewWebhook(config Config, kubeClient kubernetes.Interface, certManager certificate.Manager, meshCatalog catalog.MeshCataloger, namespaceController namespace.Controller, meshName, osmNamespace, webhookName string, stop <-chan struct{}, cfg configurator.Configurator) error {
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMControllerName, osmNamespace))
	validityPeriod := constants.XDSCertificateValidityPeriod
	cert, err := certManager.IssueCertificate(cn, &validityPeriod)
	if err != nil {
		return fmt.Errorf("Error issuing certificate for the mutating webhook: %+v", err)
	}

	wh := webhook{
		config:              config,
		kubeClient:          kubeClient,
		certManager:         certManager,
		meshCatalog:         meshCatalog,
		namespaceController: namespaceController,
		osmNamespace:        osmNamespace,
		cert:                cert,
	}

	go wh.run(stop)
	if err = patchMutatingWebhookConfiguration(cert, meshName, osmNamespace, webhookName, wh.kubeClient); err != nil {
		return fmt.Errorf("Error configuring MutatingWebhookConfiguration: %+v", err)
	}
	return nil
}

func (wh *webhook) run(stop <-chan struct{}) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.DefaultServeMux
	// HTTP handlers
	mux.HandleFunc("/health/ready", wh.healthReadyHandler)
	mux.HandleFunc(osmWebhookMutatePath, wh.mutateHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", wh.config.ListenPort),
		Handler: mux,
	}

	log.Info().Msgf("Starting sidecar-injection webhook server on :%v", wh.config.ListenPort)
	go func() error {
		// Generate a key pair from your pem-encoded cert and key ([]byte).
		cert, err := tls.X509KeyPair(wh.cert.GetCertificateChain(), wh.cert.GetPrivateKey())
		if err != nil {
			return fmt.Errorf("Error parsing webhook certificate: %+v", err)
		}

		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		if err := server.ListenAndServeTLS("", ""); err != nil {
			return fmt.Errorf("Sidecar-injection webhook HTTP server failed to start: %+v", err)
		}
		return nil
	}()

	// Wait on exit signals
	<-stop

	// Stop the server
	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error shutting down sidecar-injection webhook HTTP server")
	} else {
		log.Info().Msg("Done shutting down sidecar-injection webhook HTTP server")
	}
	return nil
}

func (wh *webhook) healthReadyHandler(w http.ResponseWriter, req *http.Request) {
	// TODO(shashank): If TLS certificate is not present, mark as not ready
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Health OK"))
	if err != nil {
		log.Error().Err(err).Msgf("Error writing bytes")
	}
}

func (wh *webhook) mutateHandler(w http.ResponseWriter, req *http.Request) {
	log.Info().Msgf("Request received: Method=%v, URL=%v", req.Method, req.URL)

	if contentType := req.Header.Get("Content-Type"); contentType != "application/json" {
		errmsg := fmt.Sprintf("Invalid Content-Type: %q", contentType)
		http.Error(w, errmsg, http.StatusUnsupportedMediaType)
		log.Error().Msgf("Request error: error=%s, code=%v", errmsg, http.StatusUnsupportedMediaType)
		return
	}

	var body []byte
	if req.Body != nil {
		var err error
		if body, err = ioutil.ReadAll(req.Body); err != nil {
			errmsg := fmt.Sprintf("Error reading request body: %s", err)
			http.Error(w, errmsg, http.StatusInternalServerError)
			log.Error().Msgf("Request error: error=%s, code=%v", errmsg, http.StatusInternalServerError)
			return
		}
	}

	if len(body) == 0 {
		errmsg := "Empty request body"
		http.Error(w, errmsg, http.StatusBadRequest)
		log.Error().Msgf("Request error: error=%s, code=%v", errmsg, http.StatusBadRequest)
		return
	}

	var admissionReq v1beta1.AdmissionReview
	var admissionResp v1beta1.AdmissionReview
	if _, _, err := deserializer.Decode(body, nil, &admissionReq); err != nil {
		log.Error().Err(err).Msg("Error decoding admission request")
		admissionResp.Response = toAdmissionError(err)
	} else {
		admissionResp.Response = wh.mutate(admissionReq.Request)
	}

	resp, err := json.Marshal(&admissionResp)
	if err != nil {
		errmsg := fmt.Sprintf("Error marshalling admission response: %s", err)
		http.Error(w, errmsg, http.StatusInternalServerError)
		log.Error().Msgf("Request error, error=%s, code=%v", errmsg, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(resp); err != nil {
		log.Error().Err(err).Msg("Error writing admission response")
	}

	log.Debug().Msg("Done responding to admission request")
}

func (wh *webhook) mutate(req *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	// Decode the Pod spec from the request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Error().Err(err).Msg("Error unmarshaling request to Pod")
		return toAdmissionError(err)
	}
	log.Info().Msgf("Mutation request: (new object: %v) (old object: %v)", string(req.Object.Raw), string(req.OldObject.Raw))

	// Start building the response
	resp := &v1beta1.AdmissionResponse{
		Allowed: true,
		UID:     req.UID,
	}

	// Check if we must inject the sidecar
	if inject, err := wh.mustInject(&pod, req.Namespace); err != nil {
		log.Error().Err(err).Msg("Error checking if sidecar must be injected")
		return toAdmissionError(err)
	} else if !inject {
		log.Info().Msg("Skipping sidecar injection")
		return resp
	}

	// Create the patches for the spec
	// We use req.Namespace because pod.Namespace is "" at this point
	patchBytes, err := wh.createPatch(&pod, req.Namespace)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create patch")
		return toAdmissionError(err)
	}

	patchAdmissionResponse(resp, patchBytes)
	log.Info().Msg("Done patching admission response")
	return resp
}

func (wh *webhook) isNamespaceAllowed(namespace string) bool {
	// Skip Kubernetes system namespaces
	for _, ns := range kubeSystemNamespaces {
		if ns == namespace {
			return false
		}
	}
	// Skip namespaces not being observed
	return wh.namespaceController.IsMonitoredNamespace(namespace)
}

// mustInject determines whether the sidecar must be injected.
//
// The sidecar injection is performed when:
// 1. The namespace is annotated for OSM monitoring, and
// 2. The POD is not annotated with sidecar-injection or is set to enabled/yes/true
//
// The sidecar injection is not performed when:
// 1. The namespace is not annotated for OSM monitoring, or
// 2. The POD is annotated with sidecar-injection set to disabled/no/false
//
// The function returns an error when:
// 1. The value of the POD level sidecar-injection annotation is invalid
func (wh *webhook) mustInject(pod *corev1.Pod, namespace string) (bool, error) {
	// If the request belongs to a namespace we are not monitoring, skip it
	if !wh.isNamespaceAllowed(namespace) {
		log.Info().Msgf("Request belongs to namespace=%s not in the list of monitored namespaces", namespace)
		return false, nil
	}

	// Check if the POD is annotated for injection
	inject := strings.ToLower(pod.ObjectMeta.Annotations[annotationInject])
	log.Debug().Msgf("Sidecar injection annotation: '%s:%s'", annotationInject, inject)
	if inject != "" {

		switch inject {
		case "enabled", "yes", "true":
			return true, nil
		case "disabled", "no", "false":
			return false, nil
		default:
			return false, fmt.Errorf("Invalid annotion value specified for annotation %q: %s", annotationInject, inject)
		}
	}

	// If we reached here, it means the namespace was annotated for OSM to monitor
	// and no POD level sidecar injection overrides are present.
	return true, nil
}

func toAdmissionError(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func patchAdmissionResponse(resp *v1beta1.AdmissionResponse, patchBytes []byte) {
	resp.Patch = patchBytes
	resp.PatchType = func() *v1beta1.PatchType {
		pt := v1beta1.PatchTypeJSONPatch
		return &pt
	}()
}

func patchMutatingWebhookConfiguration(cert certificate.Certificater, meshName, osmNamespace, webhookName string, clientSet kubernetes.Interface) error {
	if err := hookExists(clientSet, webhookName); err != nil {
		log.Error().Err(err).Msgf("Error getting webhook %s", webhookName)
	}
	updatedWH := admissionv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
		Webhooks: []admissionv1beta1.MutatingWebhook{
			{
				Name: osmWebhookName,
				ClientConfig: admissionv1beta1.WebhookClientConfig{
					CABundle: cert.GetCertificateChain(),
				},
				Rules: []admissionv1beta1.RuleWithOperations{
					{
						Operations: []admissionv1beta1.OperationType{admissionv1beta1.Create},
						Rule: admissionv1beta1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
			},
		},
	}
	data, err := json.Marshal(updatedWH)
	if err != nil {
		return err
	}

	_, err = clientSet.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Patch(
		context.Background(), webhookName, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error configuring webhook %s", webhookName)
		return err
	}

	log.Info().Msgf("Configured MutatingWebhookConfiguration %s", webhookName)
	return nil
}

func hookExists(clientSet kubernetes.Interface, webhookName string) error {
	_, err := clientSet.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.Background(), webhookName, metav1.GetOptions{})
	return err
}
