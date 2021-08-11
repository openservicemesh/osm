// Package validator implements utility routines related to Kubernetes' admission webhooks.
package validator

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/webhook"
)

var (
	defaultValidators = map[string]Validator{}

	// validationAPIPath is the API path for performing resource validations
	validationAPIPath = "/validate"

	// HealthAPIPath is the API path for health check
	HealthAPIPath = "/healthz"
)

// RegisterValidator registers all validators. It is not thread safe.
// It assumes one validator per GVK. If multiple validations need to happen it should all happen in the single validator
func RegisterValidator(gvk string, v Validator) {
	defaultValidators[gvk] = v
}

/*
There are a few ways to utilize the Validator function:

1. return resp, nil

	In this case we simply return the raw resp. This allows for the most customization.

2. return nil, err

	In this case we convert the error to an AdmissionResponse.  If the error type is an AdmissionError, we
	convert accordingly, which allows for some customization of the AdmissionResponse. Otherwise, we set Allow to
	false and the status to the error message.

3. return nil, nil

	In this case we create a simple AdmissionResponse, with Allow set to true.

4. Note that resp, err will ignore the error. It assumes that you are returning nil for resp if there is an error

In all of the above cases we always populate the UID of the response from the request.

An example of a validator:

func FakeValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	o, n := &FakeObj{}, &FakeObj{}
	// If you need to compare against the old object
	if err := json.NewDecoder(bytes.NewBuffer(req.OldObject.Raw)).Decode(o); err != nil {
		return nil, err
	}

	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(n); err != nil {
		returrn nil, err
	}

	// validate the objects, potentially returning an error, or a more detailed AdmissionResponse.

	// This will set allow to true
	return nil, nil
}
*/

// Validator is a function that accepts an AdmissionRequest and returns an AdmissionResponse.
type Validator func(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error)

// ValidatingWebhookServer implements the K8s Validating Webhook API, and runs the associated validator func.
type ValidatingWebhookServer struct {
	// Map of Resource (GroupVersionKind), to validator
	Validators map[string]Validator
}

// NewValidatingWebhook returns a ValidatingWebhookServer with the defaultValidators that were previously registered.
func NewValidatingWebhook(webhookConfigName string, port int, certificater certificate.Certificater, kubeClient kubernetes.Interface, stop <-chan struct{}) (*ValidatingWebhookServer, error) {
	vCopy := make(map[string]Validator, len(defaultValidators))
	for k, v := range defaultValidators {
		vCopy[k] = v
	}
	v := &ValidatingWebhookServer{
		Validators: vCopy,
	}
	// Update the updateValidatingWebhookConfig with the OSM CA bundle
	if err := updateValidatingWebhookCABundle(webhookConfigName, certificater, kubeClient); err != nil {
		return nil, errors.Errorf("Error configuring ValidatingWebhookConfiguration %s: %+v", webhookConfigName, err)
	}
	go v.run(port, certificater, stop)
	return v, nil
}

func (s *ValidatingWebhookServer) doValidation(w http.ResponseWriter, req *http.Request) {
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

func (s *ValidatingWebhookServer) getAdmissionReqResp(admissionRequestBody []byte) (requestForNamespace string, admissionResp admissionv1.AdmissionReview) {
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

func (s *ValidatingWebhookServer) handleValidation(req *admissionv1.AdmissionRequest) (resp *admissionv1.AdmissionResponse) {
	var err error
	defer func() {
		if resp == nil {
			resp = &admissionv1.AdmissionResponse{Allowed: true}
		}
		resp.UID = req.UID // ensure this is always set
	}()
	gvk := req.Kind.String()
	v, ok := s.Validators[gvk]
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

func (s *ValidatingWebhookServer) run(port int, certificater certificate.Certificater, stop <-chan struct{}) {
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
			// TODO: Need to push metric?
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
