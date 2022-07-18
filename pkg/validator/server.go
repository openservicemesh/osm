// Package validator implements utility routines related to Kubernetes' admission webhooks.
package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/policy"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/webhook"
)

var (
	// validationAPIPath is the API path for performing resource validations
	validationAPIPath = "/validate"
)

// validatingWebhookServer implements the K8s Validating Webhook API, and runs the associated validator func.
type validatingWebhookServer struct {
	// Map of Resource (GroupVersionKind), to validator
	validators map[string]validateFunc
}

// NewValidatingWebhook returns a validatingWebhookServer with the defaultValidators that were previously registered.
func NewValidatingWebhook(ctx context.Context, webhookConfigName, osmNamespace, osmVersion, meshName string, enableReconciler, validateTrafficTarget bool, certManager *certificate.Manager, kubeClient kubernetes.Interface, policyClient policy.Controller) error {
	kv := &policyValidator{
		policyClient: policyClient,
	}

	v := &validatingWebhookServer{
		validators: map[string]validateFunc{
			policyv1alpha1.SchemeGroupVersion.WithKind("IngressBackend").String():         kv.ingressBackendValidator,
			policyv1alpha1.SchemeGroupVersion.WithKind("Egress").String():                 egressValidator,
			policyv1alpha1.SchemeGroupVersion.WithKind("UpstreamTrafficSetting").String(): kv.upstreamTrafficSettingValidator,
			smiAccess.SchemeGroupVersion.WithKind("TrafficTarget").String():               trafficTargetValidator,
		},
	}

	srv, err := webhook.NewServer(ValidatorWebhookSvc, osmNamespace, constants.ValidatorWebhookPort, certManager, map[string]http.HandlerFunc{
		validationAPIPath: v.doValidation,
	}, func(cert *certificate.Certificate) error {
		if err := createOrUpdateValidatingWebhook(kubeClient, cert, webhookConfigName, meshName, osmNamespace, osmVersion, validateTrafficTarget, enableReconciler); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	go srv.Run(ctx)
	return nil
}

func (s *validatingWebhookServer) doValidation(w http.ResponseWriter, req *http.Request) {
	log.Trace().Msgf("Received validating webhook request: Method=%v, URL=%v", req.Method, req.URL)

	if contentType := req.Header.Get(webhook.HTTPHeaderContentType); contentType != webhook.ContentTypeJSON {
		err := fmt.Errorf("Invalid content type %s; Expected %s", contentType, webhook.ContentTypeJSON)
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

	webhook.RecordAdmissionMetrics(admissionReq.Request, admissionResp.Response)

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
