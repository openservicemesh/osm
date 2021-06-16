// Package validator implements utility routines related to Kubernetes' admission webhooks.
package validator

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/webhook"
)

var (
	defaultValidators = map[string]Validator{}

	// ValidatingWebhookPath is the path for the validating webhook.
	ValidatingWebhookPath = "/validate"
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

// HandleValidation implements the HTTP API for the Validating Webhook.
func (s *ValidatingWebhookServer) HandleValidation(w http.ResponseWriter, req *http.Request) {
	defer dclose(req.Body)
	adReq := new(admissionv1.AdmissionRequest)
	if err := json.NewDecoder(req.Body).Decode(adReq); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Error reading admission request body; Responded to admission request with HTTP %v", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(s.handleValidation(adReq))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Error marshaling admission response body; Responded to admission request with HTTP %v", http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Truncated response; error writing output to http writer; Responsed to admission request with HTTP %v", http.StatusInternalServerError)
		return
	}
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

	mux.HandleFunc(ValidatingWebhookPath, s.HandleValidation)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Info().Msgf("Starting sidecar-injection webhook server on port: %v", port)
	go func() {
		// Wait on exit signals
		<-stop

		// Stop the server
		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Error shutting down validating webhook HTTP server")
		} else {
			log.Info().Msg("Done shutting down validating webhook HTTP server")
		}
	}()
	// Generate a key pair from your pem-encoded cert and key ([]byte).
	cert, err := tls.X509KeyPair(certificater.GetCertificateChain(), certificater.GetPrivateKey())
	if err != nil {
		log.Error().Err(err).Msg("Error parsing webhook certificate")
		return
	}

	// #nosec G402
	server.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Error().Err(err).Msg("Validating webhook HTTPS server failed")
		return
	}
}

// defer closer
func dclose(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Error().Err(err).Msgf("Error closing HTTP request body. Potential memory leak.")
	}
}
