// Package webhook implements utility routines related to Kubernetes' admission webhooks.
package webhook

import (
	"io/ioutil"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("webhook")

	// ContentTypeJSON is the supported content type for HTTP requests
	ContentTypeJSON = "application/json"

	// HTTPHeaderContentType is the Content-Type HTTP header key
	HTTPHeaderContentType = "Content-Type"

	// codecs is the codec factory used by the deserialzer
	codecs = serializer.NewCodecFactory(runtime.NewScheme())

	// Deserializer is used to decode the admission request body
	Deserializer = codecs.UniversalDeserializer()
)

// GetAdmissionRequestBody returns the body of the admission request
func GetAdmissionRequestBody(w http.ResponseWriter, req *http.Request) ([]byte, error) {
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
		return admissionRequestBody, err
	}

	if len(admissionRequestBody) == 0 {
		return emptyBodyError()
	}

	return admissionRequestBody, nil
}

// AdmissionError wraps error as AdmissionResponse
func AdmissionError(err error) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}
