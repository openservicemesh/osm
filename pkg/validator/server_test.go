package validator

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type fakeObj struct {
	Allow        bool
	Error        bool
	ExplicitResp bool
}

func TestHandleValidation(t *testing.T) {
	gvk := metav1.GroupVersionKind{
		Kind:    "Fake",
		Group:   "fake.osm.io",
		Version: "v1alpha1",
	}
	s := ValidatingWebhookServer{
		Validators: map[string]Validator{
			gvk.String(): func(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
				f := fakeObj{}
				if err := json.Unmarshal(req.Object.Raw, &f); err != nil {
					return nil, err
				}
				if f.ExplicitResp {
					return &admissionv1.AdmissionResponse{
						Allowed: f.Allow,
						Result:  &metav1.Status{Message: "explicit response"},
					}, nil
				}
				if f.Error {
					return nil, errors.New("explicit error")
				}
				return nil, nil
			},
		},
	}
	badGvk := metav1.GroupVersionKind{
		Kind:    "Fake",
		Group:   "fake.osm.io",
		Version: "badVersion",
	}
	testCases := []struct {
		testName string
		req      *admissionv1.AdmissionRequest
		expResp  *admissionv1.AdmissionResponse
	}{
		{
			testName: "unknown gvk",
			req: &admissionv1.AdmissionRequest{
				UID:  "1",
				Kind: badGvk,
				Object: runtime.RawExtension{
					Raw: []byte(`{}`),
				},
			},
			expResp: &admissionv1.AdmissionResponse{
				UID:    "1",
				Result: &metav1.Status{Message: fmt.Errorf("unknown gvk: %s", badGvk).Error()},
			},
		},
		{
			testName: "invalid obj returned explicit error",
			req: &admissionv1.AdmissionRequest{
				UID:  "2",
				Kind: gvk,
				Object: runtime.RawExtension{
					Raw: []byte(`{"Error": true}`),
				},
			},
			expResp: &admissionv1.AdmissionResponse{
				UID:    "2",
				Result: &metav1.Status{Message: "explicit error"},
			},
		},
		{
			testName: "valid obj explicit response obj",
			req: &admissionv1.AdmissionRequest{
				UID:  "3",
				Kind: gvk,
				Object: runtime.RawExtension{
					Raw: []byte(`{"Allow": true, "ExplicitResp": true}`),
				},
			},
			expResp: &admissionv1.AdmissionResponse{
				UID:     "3",
				Allowed: true,
				Result:  &metav1.Status{Message: "explicit response"},
			},
		},
		{
			testName: "valid obj implicit response obj",
			req: &admissionv1.AdmissionRequest{
				UID:  "4",
				Kind: gvk,
				Object: runtime.RawExtension{
					Raw: []byte(`{}`),
				},
			},
			expResp: &admissionv1.AdmissionResponse{
				UID:     "4",
				Allowed: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			resp := s.handleValidation(tc.req)
			assert := tassert.New(t)

			assert.NotNil(resp)
			assert.Equal(tc.expResp, resp)
		})
	}
}
