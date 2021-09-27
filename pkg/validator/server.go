// Package validator implements utility routines related to Kubernetes' admission webhooks.
package validator

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/webhook"
)

var (
	// validationAPIPath is the API path for performing resource validations
	validationAPIPath = "/validate"

	// HealthAPIPath is the API path for health check
	HealthAPIPath = "/healthz"
)

// validatingWebhookServer implements the K8s Validating Webhook API, and runs the associated validator func.
type validatingWebhookServer struct {
	// Map of Resource (GroupVersionKind), to validator
	validators map[string]validateFunc
}

// NewValidatingWebhook returns a validatingWebhookServer with the defaultValidators that were previously registered.
func NewValidatingWebhook(webhookConfigName, osmNamespace, osmVersion, meshName string, enableReconciler, validateTrafficTarget bool, port int, certificater certificate.Certificater, kubeClient kubernetes.Interface, stop <-chan struct{}) error {
	v := &validatingWebhookServer{
		validators: map[string]validateFunc{
			policyv1alpha1.SchemeGroupVersion.WithKind("IngressBackend").String(): ingressBackendValidator,
			policyv1alpha1.SchemeGroupVersion.WithKind("Egress").String():         egressValidator,
			smiAccess.SchemeGroupVersion.WithKind("TrafficTarget").String():       trafficTargetValidator,
		},
	}

	if enableReconciler {
		// Create the ValidatingWebhook
		if err := createValidatingWebhook(kubeClient, certificater, webhookConfigName, meshName, osmNamespace, osmVersion, validateTrafficTarget); err != nil {
			return errors.Errorf("Error creating ValidatingWebhookConfiguration %s: %+v", webhookConfigName, err)
		}
	} else {
		// Update the updateValidatingWebhookConfig with the OSM CA bundle, as the MutatingWebhook is created via Helm
		if err := updateValidatingWebhookCABundle(webhookConfigName, certificater, kubeClient); err != nil {
			return errors.Wrapf(err, "Error configuring ValidatingWebhookConfiguration %s", webhookConfigName)
		}
	}

	go v.run(port, certificater, stop)
	return nil
}

func (s *validatingWebhookServer) doValidation(w http.ResponseWriter, req *http.Request) {
	log.Trace().Msgf("Received validating webhook request: Method=%v, URL=%v", req.Method, req.URL)

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

	requestForNamespace, admissionResp := s.getAdmissionReqResp(admissionRequestBody)

	resp, err := json.Marshal(&admissionResp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshalling admission response: %s", err), http.StatusInternalServerError)
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingKubernetesResource)).
			Msgf("Error marshalling admission response; Responded to admission request in namespace %s with HTTP %v", requestForNamespace, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(resp); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrWritingAdmissionResp)).
			Msgf("Error writing admission response for request in namespace %s", requestForNamespace)
	}

	log.Trace().Msgf("Done responding to admission request in namespace %s", requestForNamespace)
}

func (s *validatingWebhookServer) getAdmissionReqResp(admissionRequestBody []byte) (requestForNamespace string, admissionResp admissionv1.AdmissionReview) {
	var admissionReq admissionv1.AdmissionReview
	if _, _, err := webhook.Deserializer.Decode(admissionRequestBody, nil, &admissionReq); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingAdmissionReqBody)).
			Msg("Error decoding admission request body")
		admissionResp.Response = webhook.AdmissionError(err)
	} else {
		admissionResp.Response = s.handleValidation(admissionReq.Request)
	}
	admissionResp.TypeMeta = admissionReq.TypeMeta
	admissionResp.Kind = admissionReq.Kind

	return admissionReq.Request.Namespace, admissionResp
}

func (s *validatingWebhookServer) handleValidation(req *admissionv1.AdmissionRequest) (resp *admissionv1.AdmissionResponse) {
	var err error
	defer func() {
		if resp == nil {
			resp = &admissionv1.AdmissionResponse{Allowed: true}
		}
		resp.UID = req.UID // ensure this is always set
	}()
	gvk := req.Kind.String()
	v, ok := s.validators[gvk]
	if !ok {
		return webhook.AdmissionError(fmt.Errorf("unknown gvk: %s", gvk))
	}

	// We don't explicitly do an if err != nil, since we will set it from
	resp, err = v(req)
	if resp != nil {
		if err != nil {
			log.Warn().Msgf("Warning! validator for gvk: %s returned both an AdmissionResponse *and* an error. Please return one or the other", gvk)
		}
		return
	}
	// No response, but got an error.
	if err != nil {
		resp = webhook.AdmissionError(err)
	}
	return
}

func (s *validatingWebhookServer) run(port int, certificater certificate.Certificater, stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()

	mux.HandleFunc(validationAPIPath, s.doValidation)
	mux.HandleFunc(HealthAPIPath, healthHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Info().Msgf("Starting resource validator webhook server on port: %v", port)
	go func() {
		// Generate a key pair from your pem-encoded cert and key
		cert, err := tls.X509KeyPair(certificater.GetCertificateChain(), certificater.GetPrivateKey())
		if err != nil {
			// TODO: Need to push metric
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrParsingValidatingWebhookCert)).
				Msg("Error parsing webhook certificate")
			return
		}

		// #nosec G402
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		if err := server.ListenAndServeTLS("", ""); err != nil {
			// TODO(#3962): metric might not be scraped before process restart resulting from this error
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrStartingValidatingWebhookHTTPServer)).
				Msg("Resource validator webhook HTTP server failed to start")
			return
		}
	}()

	// Wait on exit signals
	<-stop

	// Stop the server
	if err := server.Shutdown(ctx); err != nil {
		// TODO: Needto push metric?
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrShuttingDownValidatingWebhookHTTPServer)).
			Msg("Error shutting down resource validator webhook HTTP server")
	} else {
		log.Info().Msg("Done shutting down resource validator webhook HTTP server")
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Health OK")); err != nil {
		log.Error().Err(err).Msg("Error writing bytes for health check response")
	}
}
