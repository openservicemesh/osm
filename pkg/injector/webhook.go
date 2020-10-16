package injector

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/pkg/errors"
	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
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
	// mutatingWebhookName is the name of the mutating webhook used for sidecar injection
	mutatingWebhookName = "osm-inject.k8s.io"

	// webhookMutatePath is the HTTP path at which the webhook exptects to receive mutation requests
	webhookMutatePath = "/mutate"

	// WebhookHealthPath is the HTTP path at which the health of the webhook can be queried
	WebhookHealthPath = "/healthz"
)

// NewWebhook starts a new web server handling requests from the injector MutatingWebhookConfiguration
func NewWebhook(config Config, kubeClient kubernetes.Interface, certManager certificate.Manager, meshCatalog catalog.MeshCataloger, kubeController k8s.Controller, meshName, osmNamespace, webhookConfigName string, stop <-chan struct{}, cfg configurator.Configurator) error {
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMControllerName, osmNamespace))
	cert, err := certManager.IssueCertificate(cn, constants.XDSCertificateValidityPeriod)
	if err != nil {
		return errors.Errorf("Error issuing certificate for the mutating webhook: %+v", err)
	}

	wh := webhook{
		config:         config,
		kubeClient:     kubeClient,
		certManager:    certManager,
		meshCatalog:    meshCatalog,
		kubeController: kubeController,
		osmNamespace:   osmNamespace,
		cert:           cert,
		configurator:   cfg,
	}

	go wh.run(stop)
	if err = patchMutatingWebhookConfiguration(cert, meshName, osmNamespace, webhookConfigName, wh.kubeClient); err != nil {
		return errors.Errorf("Error configuring MutatingWebhookConfiguration: %+v", err)
	}
	return nil
}

func (wh *webhook) run(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.DefaultServeMux
	// HTTP handlers
	mux.HandleFunc(WebhookHealthPath, wh.healthHandler)
	mux.HandleFunc(webhookMutatePath, wh.mutateHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", wh.config.ListenPort),
		Handler: mux,
	}

	log.Info().Msgf("Starting sidecar-injection webhook server on :%v", wh.config.ListenPort)
	go func() {
		// Generate a key pair from your pem-encoded cert and key ([]byte).
		cert, err := tls.X509KeyPair(wh.cert.GetCertificateChain(), wh.cert.GetPrivateKey())
		if err != nil {
			log.Error().Err(err).Msgf("Error parsing webhook certificate: %+v", err)
			return
		}

		// #nosec G402
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Error().Err(err).Msgf("Sidecar-injection webhook HTTP server failed to start: %+v", err)
			return
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

func (wh *webhook) healthHandler(w http.ResponseWriter, req *http.Request) {
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
	if req == nil {
		log.Error().Msg("Nil AdmissionRequest")
		return toAdmissionError(errors.New("nil admission request"))
	}

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
	// This string uniquely identifies the pod. Ideally this would be the pod.UID, but this is not available at this point.
	proxyUUID := uuid.New().String()
	patchBytes, err := wh.createPatch(&pod, req.Namespace, proxyUUID)
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
	return wh.kubeController.IsMonitoredNamespace(namespace)
}

// mustInject determines whether the sidecar must be injected.
//
// The sidecar injection is performed when the namespace is labeled for monitoring and either of the following is true:
// 1. The pod is explicitly annotated with enabled/yes/true for sidecar injection, or
// 2. The namespace is annotated for sidecar injection and the pod is not explicitly annotated with disabled/no/false
//
// The function returns an error when it is unable to determine whether to perform sidecar injection.
func (wh *webhook) mustInject(pod *corev1.Pod, namespace string) (bool, error) {
	// If the request belongs to a namespace we are not monitoring, skip it
	if !wh.isNamespaceAllowed(namespace) {
		log.Info().Msgf("Request belongs to namespace=%s, not in the list of monitored namespaces", namespace)
		return false, nil
	}

	// Check if the pod is annotated for injection
	podInjectAnnotationExists, podInject, err := isAnnotatedForInjection(pod.Annotations)
	if err != nil {
		log.Error().Err(err).Msg("Error determining if the pod is enabled for sidecar injection")
		return false, err
	}

	// Check if the namespace is annotated for injection
	ns := wh.kubeController.GetNamespace(namespace)
	if ns == nil {
		log.Error().Err(errNamespaceNotFound).Msgf("Error retrieving namespace %s", namespace)
		return false, err
	}
	nsInjectAnnotationExists, nsInject, err := isAnnotatedForInjection(ns.Annotations)
	if err != nil {
		log.Error().Err(err).Msg("Error determining if namespace %s is enabled for sidecar injection")
		return false, err
	}

	if podInjectAnnotationExists && podInject {
		// Pod is explicitly annotated to enable sidecar injection
		return true, nil
	} else if nsInjectAnnotationExists && nsInject {
		// Namespace is annotated to enable sidecar injection
		if !podInjectAnnotationExists || podInject {
			// If pod annotation doesn't exist or if an annotation exists to enable injection, enable it
			return true, nil
		}
	}

	// Conditions to inject the sidecar are not met
	return false, nil
}

func isAnnotatedForInjection(annotations map[string]string) (exists bool, enabled bool, err error) {
	inject := strings.ToLower(annotations[constants.SidecarInjectionAnnotation])
	log.Trace().Msgf("Sidecar injection annotation: '%s:%s'", constants.SidecarInjectionAnnotation, inject)
	if inject != "" {
		exists = true
		switch inject {
		case "enabled", "yes", "true":
			enabled = true
		case "disabled", "no", "false":
			enabled = false
		default:
			err = errors.Errorf("Invalid annotion value specified for annotation %q: %s", constants.SidecarInjectionAnnotation, inject)
		}
	}
	return
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
	pt := v1beta1.PatchTypeJSONPatch
	resp.PatchType = &pt
}

func patchMutatingWebhookConfiguration(cert certificate.Certificater, meshName, osmNamespace, webhookConfigName string, clientSet kubernetes.Interface) error {
	if err := hookExists(clientSet, webhookConfigName); err != nil {
		log.Error().Err(err).Msgf("Error getting MutatingWebhookConfiguration %s", webhookConfigName)
	}
	updatedWH := admissionv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookConfigName,
		},
		Webhooks: []admissionv1beta1.MutatingWebhook{
			{
				Name: mutatingWebhookName,
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
		context.Background(), webhookConfigName, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error configuring MutatingWebhookConfiguration %s", webhookConfigName)
		return err
	}

	log.Info().Msgf("Configured MutatingWebhookConfiguration %s", webhookConfigName)
	return nil
}

func hookExists(clientSet kubernetes.Interface, webhookConfigName string) error {
	_, err := clientSet.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.Background(), webhookConfigName, metav1.GetOptions{})
	return err
}
