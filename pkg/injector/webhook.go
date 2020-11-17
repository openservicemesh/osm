package injector

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	admissionRegistrationTypes "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

var (
	codecs       = serializer.NewCodecFactory(runtime.NewScheme())
	deserializer = codecs.UniversalDeserializer()
)

const (
	// MutatingWebhookName is the name of the mutating webhook used for sidecar injection
	MutatingWebhookName = "osm-inject.k8s.io"

	// webhookCreatePod is the HTTP path at which the webhook expects to receive pod creation events
	webhookCreatePod = "/mutate-pod-creation"

	// WebhookHealthPath is the HTTP path at which the health of the webhook can be queried
	WebhookHealthPath = "/healthz"

	// webhookTimeoutStr is the url variable name for timeout
	webhookMutateTimeoutKey = "timeout"

	contentTypeJSON       = "application/json"
	httpHeaderContentType = "Content-Type"
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

		// Envoy sidecars should never be injected in these namespaces
		nonInjectNamespaces: mapset.NewSetFromSlice([]interface{}{
			metav1.NamespaceSystem,
			metav1.NamespacePublic,
			osmNamespace,
		}),
	}

	// Start the MutatingWebhook web server
	go wh.run(stop)

	// Update the MutatingWebhookConfig with the OSM CA bundle
	if err = updateMutatingWebhookCABundle(cert, webhookConfigName, wh.kubeClient); err != nil {
		return errors.Errorf("Error configuring MutatingWebhookConfiguration %s: %+v", webhookConfigName, err)
	}
	return nil
}

func (wh *webhook) run(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.DefaultServeMux

	mux.HandleFunc(WebhookHealthPath, healthHandler)

	// We know that the events arriving at this handler are CREATE POD only
	// because of the specifics of MutatingWebhookConfiguration template in this repository.
	mux.HandleFunc(webhookCreatePod, wh.podCreationHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", wh.config.ListenPort),
		Handler: mux,
	}

	log.Info().Msgf("Starting sidecar-injection webhook server on port: %v", wh.config.ListenPort)
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

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Health OK")); err != nil {
		log.Error().Err(err).Msgf("Error writing bytes for mutating webhook health check handler")
	}
}

func (wh *webhook) getAdmissionReqResp(proxyUUID uuid.UUID, admissionRequestBody []byte) (requestForNamespace string, admissionResp v1beta1.AdmissionReview) {
	var admissionReq v1beta1.AdmissionReview
	if _, _, err := deserializer.Decode(admissionRequestBody, nil, &admissionReq); err != nil {
		log.Error().Err(err).Msg("Error decoding admission request body")
		admissionResp.Response = admissionError(err)
	} else {
		admissionResp.Response = wh.mutate(admissionReq.Request, proxyUUID)
	}

	return admissionReq.Request.Namespace, admissionResp
}

func getAdmissionRequestBody(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	emptyBodyError := func() ([]byte, error) {
		http.Error(w, errEmptyAdmissionRequestBody.Error(), http.StatusBadRequest)
		log.Error().Err(errEmptyAdmissionRequestBody).Msgf("Responded to admission request with HTTP %v", http.StatusBadRequest)
		return nil, errEmptyAdmissionRequestBody
	}

	if req.Body == nil {
		return emptyBodyError()
	}

	admissionRequestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Error reading admission request body; Responded to admission request with HTTP %v", http.StatusInternalServerError)
		return admissionRequestBody, nil
	}

	if len(admissionRequestBody) == 0 {
		return emptyBodyError()
	}

	return admissionRequestBody, nil
}

// podCreationHandler is a MutatingWebhookConfiguration handler exclusive to POD CREATE events.
func (wh *webhook) podCreationHandler(w http.ResponseWriter, req *http.Request) {
	log.Trace().Msgf("Received mutating webhook request: Method=%v, URL=%v", req.Method, req.URL)

	// For debug/profiling purposes
	if log.GetLevel() == zerolog.DebugLevel {
		// Read timeout from request
		reqTimeout, err := readTimeout(req)
		if err != nil {
			log.Error().Msgf("Could not read timeout from request url: %v", err)
		} else {
			defer webhookTimeTrack(time.Now(), *reqTimeout)
		}
	}

	if contentType := req.Header.Get(httpHeaderContentType); contentType != contentTypeJSON {
		err := errors.Errorf("Invalid content type %s; Expected %s", contentType, contentTypeJSON)
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
		log.Error().Err(err).Msgf("Responded to admission request with HTTP %v", http.StatusUnsupportedMediaType)
		return
	}

	admissionRequestBody, err := getAdmissionRequestBody(w, req)
	if err != nil {
		// Error was already logged and written to the ResponseWriter
		return
	}

	// Create the patches for the spec
	// We use req.Namespace because pod.Namespace is "" at this point
	// This string uniquely identifies the pod. Ideally this would be the pod.UID, but this is not available at this point.
	proxyUUID := uuid.New()

	requestForNamespace, admissionResp := wh.getAdmissionReqResp(proxyUUID, admissionRequestBody)

	resp, err := json.Marshal(&admissionResp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshalling admission response: %s", err), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Error marshalling admission response; Responded to admission request for pod with UUID %s in namespace %s with HTTP %v", proxyUUID, requestForNamespace, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(resp); err != nil {
		log.Error().Err(err).Msgf("Error writing admission response for pod with UUID %s in namespace %s", proxyUUID, requestForNamespace)
	}

	log.Trace().Msgf("Done responding to admission request for pod with UUID %s in namespace %s", proxyUUID, requestForNamespace)
}

func (wh *webhook) mutate(req *v1beta1.AdmissionRequest, proxyUUID uuid.UUID) *v1beta1.AdmissionResponse {
	if req == nil {
		log.Error().Msg("nil admission Request")
		return admissionError(errNilAdmissionRequest)
	}

	// Decode the Pod spec from the request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Error().Err(err).Msgf("Error unmarshaling request to pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return admissionError(err)
	}
	log.Trace().Msgf("Mutation request: (new object: %v) (old object: %v)", string(req.Object.Raw), string(req.OldObject.Raw))

	// Start building the response
	resp := &v1beta1.AdmissionResponse{
		Allowed: true,
		UID:     req.UID,
	}

	// Check if we must inject the sidecar
	if inject, err := wh.mustInject(&pod, req.Namespace); err != nil {
		log.Error().Err(err).Msgf("Error checking if sidecar must be injected for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return admissionError(err)
	} else if !inject {
		log.Trace().Msgf("Skipping sidecar injection for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return resp
	}

	patchBytes, err := wh.createPatch(&pod, req.Namespace, proxyUUID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create patch for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return admissionError(err)
	}

	patchAdmissionResponse(resp, patchBytes)
	log.Trace().Msgf("Done creating patch admission response for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
	return resp
}

func (wh *webhook) isNamespaceInjectable(namespace string) bool {
	// Never inject pods in the OSM Controller namespace or kube-public or kube-system
	isInjectableNS := !wh.nonInjectNamespaces.Contains(namespace)

	// Ignore namespaces not joined in the mesh.
	return isInjectableNS && wh.kubeController.IsMonitoredNamespace(namespace)
}

// mustInject determines whether the sidecar must be injected.
//
// The sidecar injection is performed when the namespace is labeled for monitoring and either of the following is true:
// 1. The pod is explicitly annotated with enabled/yes/true for sidecar injection, or
// 2. The namespace is annotated for sidecar injection and the pod is not explicitly annotated with disabled/no/false
//
// The function returns an error when it is unable to determine whether to perform sidecar injection.
func (wh *webhook) mustInject(pod *corev1.Pod, namespace string) (bool, error) {
	if !wh.isNamespaceInjectable(namespace) {
		log.Warn().Msgf("Request is for pod with UUID %s in namespace %s; Injection in namespace %s is not permitted", namespace, pod.Name, namespace)
		return false, nil
	}

	// Check if the pod is annotated for injection
	podInjectAnnotationExists, podInject, err := isAnnotatedForInjection(pod.Annotations, pod.Kind, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
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
	nsInjectAnnotationExists, nsInject, err := isAnnotatedForInjection(ns.Annotations, ns.Kind, ns.Name)
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

func isAnnotatedForInjection(annotations map[string]string, objectKind string, objectName string) (exists bool, enabled bool, err error) {
	inject := strings.ToLower(annotations[constants.SidecarInjectionAnnotation])
	log.Trace().Msgf("%s %s has sidecar injection annotation: '%s:%s'", objectKind, objectName, constants.SidecarInjectionAnnotation, inject)
	if inject != "" {
		exists = true
		switch inject {
		case "enabled", "yes", "true":
			enabled = true
		case "disabled", "no", "false":
			enabled = false
		default:
			err = errors.Errorf("Invalid annotation value for key %q: %s", constants.SidecarInjectionAnnotation, inject)
		}
	}
	return
}

func admissionError(err error) *v1beta1.AdmissionResponse {
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

// getPartialMutatingWebhookConfiguration returns only the portion of the MutatingWebhookConfiguration that needs to be updated.
func getPartialMutatingWebhookConfiguration(cert certificate.Certificater, webhookConfigName string) admissionv1beta1.MutatingWebhookConfiguration {
	return admissionv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookConfigName,
		},
		Webhooks: []admissionv1beta1.MutatingWebhook{
			{
				Name: MutatingWebhookName,
				ClientConfig: admissionv1beta1.WebhookClientConfig{
					CABundle: cert.GetCertificateChain(),
				},
			},
		},
	}
}

// updateMutatingWebhookCABundle updates the existing MutatingWebhookConfiguration with the CA this OSM instance runs with.
// It is necessary to perform this patch because the original MutatingWebhookConfig YAML does not contain the root certificate.
func updateMutatingWebhookCABundle(cert certificate.Certificater, webhookName string, clientSet kubernetes.Interface) error {
	mwc := clientSet.AdmissionregistrationV1beta1().MutatingWebhookConfigurations()
	if err := webhookExists(mwc, webhookName); err != nil {
		log.Error().Err(err).Msgf("Error getting MutatingWebhookConfiguration %s; Will not update CA Bundle for webhook", webhookName)
	}

	patchJSON, err := json.Marshal(getPartialMutatingWebhookConfiguration(cert, webhookName))
	if err != nil {
		return err
	}

	if _, err = mwc.Patch(context.Background(), webhookName, types.StrategicMergePatchType, patchJSON, metav1.PatchOptions{}); err != nil {
		log.Error().Err(err).Msgf("Error updating CA Bundle for MutatingWebhookConfiguration %s", webhookName)
		return err
	}

	log.Info().Msgf("Finished updating CA Bundle for MutatingWebhookConfiguration %s", webhookName)
	return nil
}

func webhookExists(mwc admissionRegistrationTypes.MutatingWebhookConfigurationInterface, webhookName string) error {
	_, err := mwc.Get(context.Background(), webhookName, metav1.GetOptions{})
	return err
}
