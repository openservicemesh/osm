package webhook

import (
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog/log"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
func AdmissionError(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}
