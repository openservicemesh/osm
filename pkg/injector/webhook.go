package injector

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/webhook"
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

	// outboundPortExclusionListAnnotation is the annotation used for outbound port exclusions
	outboundPortExclusionListAnnotation = "openservicemesh.io/outbound-port-exclusion-list"

	// inboundPortExclusionListAnnotation is the annotation used for inbound port exclusions
	inboundPortExclusionListAnnotation = "openservicemesh.io/inbound-port-exclusion-list"
)

// NewMutatingWebhook starts a new web server handling requests from the injector MutatingWebhookConfiguration
func NewMutatingWebhook(config Config, kubeClient kubernetes.Interface, certManager certificate.Manager, kubeController k8s.Controller, meshName, osmNamespace, webhookConfigName, osmVersion string, webhookTimeout int32, enableReconciler bool, stop <-chan struct{}, cfg configurator.Configurator) error {
	// This is a certificate issued for the webhook handler
	// This cert does not have to be related to the Envoy certs, but it does have to match
	// the cert provisioned with the MutatingWebhookConfiguration
	webhookHandlerCert, err := certManager.IssueCertificate(
		certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMInjectorName, osmNamespace)),
		constants.XDSCertificateValidityPeriod)
	if err != nil {
		return errors.Errorf("Error issuing certificate for the mutating webhook: %+v", err)
	}

	// The following function ensures to atomically create or get the certificate from Kubernetes
	// secret API store. Multiple instances should end up with the same webhookHandlerCert after this function executed.
	webhookHandlerCert, err = providers.GetCertificateFromSecret(osmNamespace, constants.WebhookCertificateSecretName, webhookHandlerCert, kubeClient)
	if err != nil {
		return errors.Errorf("Error fetching webhook certificate from k8s secret: %s", err)
	}

	wh := mutatingWebhook{
		config:         config,
		kubeClient:     kubeClient,
		certManager:    certManager,
		kubeController: kubeController,
		osmNamespace:   osmNamespace,
		meshName:       meshName,
		cert:           webhookHandlerCert,
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

	if enableReconciler {
		// Create the MutatingWebhook
		if err = createMutatingWebhook(wh.kubeClient, webhookHandlerCert, webhookTimeout, webhookConfigName, meshName, osmNamespace, osmVersion); err != nil {
			return errors.Errorf("Error creating MutatingWebhookConfiguration %s: %+v", webhookConfigName, err)
		}
	} else {
		// Update the MutatingWebhookConfig with the OSM CA bundle only, as the MutatingWebhook is created via Helm
		if err = updateMutatingWebhookCABundle(webhookHandlerCert, webhookConfigName, wh.kubeClient); err != nil {
			return errors.Errorf("Error configuring MutatingWebhookConfiguration %s: %+v", webhookConfigName, err)
		}
	}
	return nil
}

func (wh *mutatingWebhook) run(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()

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
			// TODO(#3962): metric might not be scraped before process restart resulting from this error
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrParsingMutatingWebhookCert)).
				Msg("Error parsing webhook certificate")
			return
		}

		// #nosec G402
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		if err := server.ListenAndServeTLS("", ""); err != nil {
			// TODO(#3962): metric might not be scraped before process restart resulting from this error
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrStartingInjectionWebhookHTTPServer)).
				Msg("Sidecar injection webhook HTTP server failed to start")
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
		log.Error().Err(err).Msg("Error writing bytes for mutating webhook health check handler")
	}
}

func (wh *mutatingWebhook) getAdmissionReqResp(proxyUUID uuid.UUID, admissionRequestBody []byte) (requestForNamespace string, admissionResp admissionv1.AdmissionReview) {
	var admissionReq admissionv1.AdmissionReview
	if _, _, err := webhook.Deserializer.Decode(admissionRequestBody, nil, &admissionReq); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingAdmissionReqBody)).
			Msg("Error decoding admission request body")
		admissionResp.Response = webhook.AdmissionError(err)
	} else {
		admissionResp.Response = wh.mutate(admissionReq.Request, proxyUUID)
	}
	admissionResp.TypeMeta = admissionReq.TypeMeta
	admissionResp.Kind = admissionReq.Kind

	if admissionReq.Request != nil {
		requestForNamespace = admissionReq.Request.Namespace
	}

	return
}

// podCreationHandler is a MutatingWebhookConfiguration handler exclusive to POD CREATE events.
func (wh *mutatingWebhook) podCreationHandler(w http.ResponseWriter, req *http.Request) {
	log.Trace().Msgf("Received mutating webhook request: Method=%v, URL=%v", req.Method, req.URL)

	// Tracks the success of the current injector webhook request
	success := false
	// Read timeout from request url
	reqTimeout, err := readTimeout(req)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrParsingReqTimeout)).
			Msg("Could not read timeout from request url")
	}
	// Execute at return of this handler
	defer webhookTimeTrack(time.Now(), reqTimeout, &success)

	if contentType := req.Header.Get(webhook.HTTPHeaderContentType); contentType != webhook.ContentTypeJSON {
		err := errors.Errorf("Invalid content type %s; Expected %s", contentType, webhook.ContentTypeJSON)
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidAdmissionReqHeader)).
			Msgf("Responded to admission request with HTTP %v", http.StatusUnsupportedMediaType)
		return
	}

	admissionRequestBody, err := webhook.GetAdmissionRequestBody(w, req)
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
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingKubernetesResource)).
			Msgf("Error marshalling admission response; Responded to admission request for pod with UUID %s in namespace %s with HTTP %v", proxyUUID, requestForNamespace, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(resp); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrWritingAdmissionResp)).
			Msgf("Error writing admission response for pod with UUID %s in namespace %s", proxyUUID, requestForNamespace)
	} else {
		success = true // read by the deferred function
	}

	log.Trace().Msgf("Done responding to admission request for pod with UUID %s in namespace %s", proxyUUID, requestForNamespace)
}

func (wh *mutatingWebhook) mutate(req *admissionv1.AdmissionRequest, proxyUUID uuid.UUID) *admissionv1.AdmissionResponse {
	if req == nil {
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrNilAdmissionReq)).Msg("nil admission Request")
		return webhook.AdmissionError(errNilAdmissionRequest)
	}

	// Decode the Pod spec from the request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUnmarshallingKubernetesResource)).
			Msgf("Error unmarshaling request to pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return webhook.AdmissionError(err)
	}

	// Start building the response
	resp := &admissionv1.AdmissionResponse{
		Allowed: true,
		UID:     req.UID,
	}

	// Check if we must inject the sidecar
	if inject, err := wh.mustInject(&pod, req.Namespace); err != nil {
		log.Error().Err(err).Msgf("Error checking if sidecar must be injected for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return webhook.AdmissionError(err)
	} else if !inject {
		log.Trace().Msgf("Skipping sidecar injection for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return resp
	}

	patchBytes, err := wh.createPatch(&pod, req, proxyUUID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create patch for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return webhook.AdmissionError(err)
	}

	patchAdmissionResponse(resp, patchBytes)
	log.Trace().Msgf("Done creating patch admission response for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
	return resp
}

func (wh *mutatingWebhook) isNamespaceInjectable(namespace string) bool {
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
func (wh *mutatingWebhook) mustInject(pod *corev1.Pod, namespace string) (bool, error) {
	if !wh.isNamespaceInjectable(namespace) {
		log.Warn().Msgf("Mutation request is for pod with UID %s; Injection in Namespace %s is not permitted", pod.ObjectMeta.UID, namespace)
		return false, nil
	}

	// Check if the pod is annotated for injection
	podInjectAnnotationExists, podInject, err := isAnnotatedForInjection(pod.Annotations, "Pod", fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDeterminingPodInjectionEnablement)).
			Msg("Error determining if the pod is enabled for sidecar injection")
		return false, err
	}

	// Check if the namespace is annotated for injection
	ns := wh.kubeController.GetNamespace(namespace)
	if ns == nil {
		log.Error().Err(errNamespaceNotFound).Msgf("Error retrieving namespace %s", namespace)
		return false, errNamespaceNotFound
	}
	nsInjectAnnotationExists, nsInject, err := isAnnotatedForInjection(ns.Annotations, "Namespace", ns.Name)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDeterminingNamespaceInjectionEnablement)).
			Msgf("Error determining if namespace %s is enabled for sidecar injection", namespace)
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

// getPortExclusionListForPod gets a list of ports to exclude from sidecar traffic interception for the given
// pod and annotation kind.
//
// Ports are excluded from sidecar interception when the pod is explicitly annotated with a single or
// comma separate list of ports.
//
// The kind of exclusion (inbound vs outbound) is determined by the specified annotation.
//
// The function returns an error when it is unable to determine whether ports need to be excluded from outbound sidecar interception.
func (wh *mutatingWebhook) getPortExclusionListForPod(pod *corev1.Pod, namespace string, annotation string) ([]int, error) {
	var ports []int
	// Check if the pod is annotated for outbound port exclusion
	ports, err := isAnnotatedForPortExclusion(pod.Annotations, annotation, pod.Kind, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDeterminingPodPortExclusions)).
			Msgf("Error determining port exclusions for annotation %s on pod %s/%s", annotation, namespace, pod.Name)
		return ports, err
	}

	return ports, nil
}

func isAnnotatedForInjection(annotations map[string]string, objectKind string, objectName string) (exists bool, enabled bool, err error) {
	inject, ok := annotations[constants.SidecarInjectionAnnotation]
	if !ok {
		return
	}

	log.Trace().Msgf("%s '%s' has sidecar injection annotation: '%s:%s'", objectKind, objectName, constants.SidecarInjectionAnnotation, inject)
	exists = true
	switch strings.ToLower(inject) {
	case "enabled", "yes", "true":
		enabled = true
	case "disabled", "no", "false":
		enabled = false
	default:
		err = errors.Errorf("Invalid annotation value for key %q: %s", constants.SidecarInjectionAnnotation, inject)
	}
	return
}

func isAnnotatedForPortExclusion(annotations map[string]string, portAnnotation string, objectKind string, objectName string) (ports []int, err error) {
	portsToExcludeStr, ok := annotations[portAnnotation]
	if !ok {
		return ports, err
	}

	log.Trace().Msgf("%s %s has port exclusion annotation: '%s:%s'", objectKind, objectName, portAnnotation, portsToExcludeStr)
	portsToExclude := strings.Split(portsToExcludeStr, ",")
	for _, portStr := range portsToExclude {
		portStr := strings.TrimSpace(portStr)
		portInt, ok := strconv.Atoi(portStr)
		if ok != nil || portInt <= 0 {
			err = errors.Errorf("Invalid port '%s' specified for annotation '%s'", portStr, portAnnotation)
			ports = nil
			return ports, err
		}
		ports = append(ports, portInt)
	}
	return ports, err
}

func patchAdmissionResponse(resp *admissionv1.AdmissionResponse, patchBytes []byte) {
	resp.Patch = patchBytes
	pt := admissionv1.PatchTypeJSONPatch
	resp.PatchType = &pt
}

// getPartialMutatingWebhookConfiguration returns only the portion of the MutatingWebhookConfiguration that needs to be updated.
func getPartialMutatingWebhookConfiguration(cert certificate.Certificater, webhookConfigName string) admissionregv1.MutatingWebhookConfiguration {
	return admissionregv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookConfigName,
		},
		Webhooks: []admissionregv1.MutatingWebhook{
			{
				Name: MutatingWebhookName,
				ClientConfig: admissionregv1.WebhookClientConfig{
					CABundle: cert.GetCertificateChain(),
				},
				SideEffects: func() *admissionregv1.SideEffectClass {
					sideEffect := admissionregv1.SideEffectClassNoneOnDryRun
					return &sideEffect
				}(),
				AdmissionReviewVersions: []string{"v1"},
			},
		},
	}
}

// updateMutatingWebhookCABundle updates the existing MutatingWebhookConfiguration with the CA this OSM instance runs with.
// It is necessary to perform this patch because the original MutatingWebhookConfig YAML does not contain the root certificate.
func updateMutatingWebhookCABundle(cert certificate.Certificater, webhookName string, clientSet kubernetes.Interface) error {
	mwc := clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations()

	patchJSON, err := json.Marshal(getPartialMutatingWebhookConfiguration(cert, webhookName))
	if err != nil {
		return err
	}

	if _, err = mwc.Patch(context.Background(), webhookName, types.StrategicMergePatchType, patchJSON, metav1.PatchOptions{}); err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUpdatingMutatingWebhookCABundle)).
			Msgf("Error updating CA Bundle for MutatingWebhookConfiguration %s", webhookName)
		return err
	}

	log.Info().Msgf("Finished updating CA Bundle for MutatingWebhookConfiguration %s", webhookName)
	return nil
}

func createMutatingWebhook(clientSet kubernetes.Interface, cert certificate.Certificater, webhookTimeout int32, webhookName, meshName, osmNamespace, osmVersion string) error {
	webhookPath := webhookCreatePod
	webhookPort := int32(constants.InjectorWebhookPort)
	failuerPolicy := admissionregv1.Fail
	matchPolict := admissionregv1.Exact

	mwhc := admissionregv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
			Labels: map[string]string{
				constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
				constants.OSMAppInstanceLabelKey: meshName,
				constants.OSMAppVersionLabelKey:  osmVersion,
				"app":                            constants.OSMInjectorName,
				constants.ReconcileLabel:         strconv.FormatBool(true)}},
		Webhooks: []admissionregv1.MutatingWebhook{
			{
				Name: MutatingWebhookName,
				ClientConfig: admissionregv1.WebhookClientConfig{
					Service: &admissionregv1.ServiceReference{
						Namespace: osmNamespace,
						Name:      constants.OSMInjectorName,
						Path:      &webhookPath,
						Port:      &webhookPort,
					},
					CABundle: cert.GetCertificateChain()},
				FailurePolicy: &failuerPolicy,
				MatchPolicy:   &matchPolict,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						constants.OSMKubeResourceMonitorAnnotation: meshName,
					},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      constants.IgnoreLabel,
							Operator: metav1.LabelSelectorOpDoesNotExist,
						},
						{
							Key:      "name",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{osmNamespace},
						},
						{
							Key:      "control-plane",
							Operator: metav1.LabelSelectorOpDoesNotExist,
						},
					},
				},
				Rules: []admissionregv1.RuleWithOperations{
					{
						Operations: []admissionregv1.OperationType{admissionregv1.Create},
						Rule: admissionregv1.Rule{
							APIGroups:   []string{"*"},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
				SideEffects: func() *admissionregv1.SideEffectClass {
					sideEffect := admissionregv1.SideEffectClassNoneOnDryRun
					return &sideEffect
				}(),
				TimeoutSeconds:          &webhookTimeout,
				AdmissionReviewVersions: []string{"v1"}}},
	}

	if _, err := clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), &mwhc, metav1.CreateOptions{}); err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingMutatingWebhook)).
			Msgf("Error creating MutatingWebhookConfiguration %s", webhookName)
		return err
	}

	log.Info().Msgf("Finished creating MutatingWebhookConfiguration %s", webhookName)
	return nil
}
