package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/policy"

	"github.com/openservicemesh/osm/pkg/webhook"
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
	s := validatingWebhookServer{
		validators: map[string]validateFunc{
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
					return nil, fmt.Errorf("explicit error")
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

func TestNewValidatingWebhook(t *testing.T) {
	testNamespace := "test-namespace"
	testMeshName := "test-mesh"
	testVersion := "test-version"
	enableReconciler := false
	validateTrafficTarget := true
	t.Run("successful startup", func(t *testing.T) {
		certManager := tresorFake.NewFake(nil, 1*time.Hour)

		stop := make(chan struct{})
		defer close(stop)
		broker := messaging.NewBroker(stop)
		webhook := &admissionregv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-webhook",
			},
		}

		kube := fake.NewSimpleClientset(webhook)
		informerCollection, err := informers.NewInformerCollection("osm", stop, informers.WithKubeClient(kube))
		tassert.NoError(t, err)
		policyClient := policy.NewPolicyController(informerCollection, nil, broker)
		ctx, cancel := context.WithCancel(context.Background())
		err = NewValidatingWebhook(ctx, webhook.Name, testNamespace, testVersion, testMeshName, enableReconciler, validateTrafficTarget, certManager, kube, policyClient)
		tassert.NoError(t, err)
		cancel()
	})

	t.Run("successful startup with reconciler enabled and traffic target validation enabled", func(t *testing.T) {
		certManager := tresorFake.NewFake(nil, 1*time.Hour)
		enableReconciler = true

		stop := make(chan struct{})
		defer close(stop)
		broker := messaging.NewBroker(stop)
		kube := fake.NewSimpleClientset()
		informerCollection, err := informers.NewInformerCollection("osm", stop, informers.WithKubeClient(kube))
		tassert.NoError(t, err)
		policyClient := policy.NewPolicyController(informerCollection, nil, broker)

		err = NewValidatingWebhook(context.Background(), "my-webhook", testNamespace, testVersion, testMeshName, enableReconciler, validateTrafficTarget, certManager, kube, policyClient)
		tassert.NoError(t, err)
	})

	t.Run("successful startup with reconciler enabled and validation for traffic target disabled", func(t *testing.T) {
		certManager := tresorFake.NewFake(nil, 1*time.Hour)
		enableReconciler = true
		validateTrafficTarget = false

		stop := make(chan struct{})
		defer close(stop)
		broker := messaging.NewBroker(stop)

		kube := fake.NewSimpleClientset()
		informerCollection, err := informers.NewInformerCollection("osm", stop, informers.WithKubeClient(kube))
		tassert.NoError(t, err)
		policyClient := policy.NewPolicyController(informerCollection, nil, broker)

		err = NewValidatingWebhook(context.Background(), "my-webhook", testNamespace, testVersion, testMeshName, enableReconciler, validateTrafficTarget, certManager, kube, policyClient)
		tassert.NoError(t, err)
	})
}

func TestDoValidation(t *testing.T) {
	tests := []struct {
		name                 string
		req                  *http.Request
		expectedResponseCode int
	}{
		{
			name: "bad Content-Type",
			req: &http.Request{
				Header: map[string][]string{
					webhook.HTTPHeaderContentType: {"not-" + webhook.ContentTypeJSON},
				},
			},
			expectedResponseCode: http.StatusUnsupportedMediaType,
		},
		{
			name: "error reading request body",
			req: &http.Request{
				Header: map[string][]string{
					webhook.HTTPHeaderContentType: {webhook.ContentTypeJSON},
				},
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			name: "successful response",
			req: &http.Request{
				Header: map[string][]string{
					webhook.HTTPHeaderContentType: {webhook.ContentTypeJSON},
				},
				Body: io.NopCloser(strings.NewReader(`{
				"metadata": {
					"uid": "some-uid"
				},
				"request": {}
			}`)),
			},
			expectedResponseCode: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			s := &validatingWebhookServer{
				validators: map[string]validateFunc{},
			}
			s.doValidation(w, test.req)
			res := w.Result()
			tassert.Equal(t, test.expectedResponseCode, res.StatusCode)
		})
	}
}
