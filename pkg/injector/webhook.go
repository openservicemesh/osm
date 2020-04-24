package injector

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/open-service-mesh/osm/demo/cmd/common"
	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
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

// NewWebhook returns a new webhook object
func NewWebhook(config Config, kubeConfig *rest.Config, certManager certificate.Manager, meshCatalog catalog.MeshCataloger, namespaceController namespace.Controller, osmID, osmNamespace, webhookName string, stop <-chan struct{}) error {
	if webhookName == "" {
		return ErrInvalidWebhookName
	}
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.AggregatedDiscoveryServiceName, osmNamespace))
	cert, err := certManager.IssueCertificate(cn)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing certificate for the mutating webhook")
		return err
	}

	wh := webhook{
		config:              config,
		kubeClient:          kubernetes.NewForConfigOrDie(kubeConfig),
		certManager:         certManager,
		meshCatalog:         meshCatalog,
		namespaceController: namespaceController,
		osmNamespace:        osmNamespace,
		cert:                cert,
	}

	go wh.run(stop)

	// --- MutatingWebhookConfiguration ---
	// Generally we have accepted the principle that one-time setup and configuration will be applied via Helm.
	// The block below is exception. We start the web server and then apply the webhook configuration.
	// CA bundle is easily available via the certManager already initialized.
	// Requirement: ADS pod must have sufficient privileges to apply MutatingWebhookConfiguration.
	{
		err = createMutatingWebhookConfiguration(cert.GetIssuingCA(), osmID, osmNamespace, webhookName)
		if err != nil {
			log.Error().Err(err).Msg("Error creating MutatingWebhookConfiguration")
			return err
		}
	}

	return nil
}

func (wh *webhook) run(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.DefaultServeMux
	// HTTP handlers
	mux.HandleFunc("/health/ready", wh.healthReadyHandler)
	mux.HandleFunc("/mutate", wh.mutateHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", wh.config.ListenPort),
		Handler: mux,
	}

	log.Info().Msgf("Starting sidecar-injection webhook server on :%v", wh.config.ListenPort)
	go func() {
		if wh.config.EnableTLS {
			// Generate a key pair from your pem-encoded cert and key ([]byte).
			cert, err := tls.X509KeyPair(wh.cert.GetCertificateChain(), wh.cert.GetPrivateKey())
			if err != nil {
				// TODO(draychev): bubble these up as errors instead of fataling here (https://github.com/open-service-mesh/osm/issues/534)
				log.Fatal().Err(err).Msg("Error parsing webhook certificate")
			}

			server.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}

			if err := server.ListenAndServeTLS("", ""); err != nil {
				// TODO(draychev): bubble these up as errors instead of fataling here (https://github.com/open-service-mesh/osm/issues/534)
				log.Fatal().Err(err).Msgf("Sidecar-injection webhook HTTP server failed to start")
			}
		} else {
			if err := server.ListenAndServe(); err != nil {
				log.Fatal().Err(err).Msgf("Sidecar-injection webhook HTTP server failed to start")
			}
		}
	}()

	// Wait on exit signals
	<-stop

	// Stop the server
	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error shutting down sidecar-injection webhook HTTP server")
	} else {
		log.Info().Msg("Done shutting down sidecar-injection webhook HTTP server")
	}
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
	log.Info().Msgf("Mutation request:\nobject: %v\nold object: %v", string(req.Object.Raw), string(req.OldObject.Raw))

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

func createMutatingWebhookConfiguration(ca []byte, osmID, osmNamespace, webhookName string) error {
	log.Info().Msgf("%s is creating a webhook with name %s", constants.AggregatedDiscoveryServiceName, webhookName)
	fail := admissionv1beta1.Fail
	webhook := admissionv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: osmNamespace,
			Labels: map[string]string{
				// TODO(draychev): get this from CLI arg instead of const (https://github.com/open-service-mesh/osm/issues/542)
				"app": constants.AggregatedDiscoveryServiceName,
			},
		},
		Webhooks: []admissionv1beta1.MutatingWebhook{
			{
				Name: "osm-inject.k8s.io",
				ClientConfig: admissionv1beta1.WebhookClientConfig{
					Service: &admissionv1beta1.ServiceReference{
						Namespace: osmNamespace,
						// TODO(draychev): get this from CLI arg instead of const (https://github.com/open-service-mesh/osm/issues/542)
						Name: constants.AggregatedDiscoveryServiceName,
						Path: to.StringPtr("/mutate"),
					},
					CABundle: ca,
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
				FailurePolicy: &fail,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						namespace.MonitorLabel: osmID,
					},
				},
			},
		},
	}

	clientSet := common.GetClient()

	opts := metav1.DeleteOptions{
		GracePeriodSeconds: to.Int64Ptr(0),
	}

	webhooks, err := clientSet.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error listing webhooks, while looking for %s", webhookName)
		return err
	}

	if exists(webhookName, webhooks) {
		if err := clientSet.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(context.Background(), webhookName, opts); err != nil {
			log.Error().Err(err).Msgf("Error deleting webhook %s", webhookName)
			return err
		}
	}

	if _, err := clientSet.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.Background(), &webhook, metav1.CreateOptions{}); err != nil {
		return err
	}

	log.Info().Msgf("Created MutatingWebhookConfiguration %s in namespace %s", webhookName, osmNamespace)
	return nil
}

func exists(webhookName string, list *admissionv1beta1.MutatingWebhookConfigurationList) bool {
	for _, wh := range list.Items {
		if wh.Name == webhookName {
			return true
		}
	}
	return false
}
