package configurator

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
)

var (
	whc = &webhookConfig{
		kubeClient:   fake.NewSimpleClientset(),
		osmNamespace: "-osm-namespace-",
	}
)

const (
	fakeField            = "field"
	admissionRequestBody = `{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1",
		"request": {
		  "uid": "11111111-2222-3333-4444-555555555555",
		  "kind": {
			"group": "",
			"version": "v1",
			"kind": "ConfigMap"
		  },
		  "resource": {
			"group": "",
			"version": "v1",
			"resource": "configmaps"
		  },
		  "name": "some-config-map",
		  "namespace": "default",
		  "oldObject": null,
		  "dryRun": false,
		  "options": null
		}
	  }`
)

func TestNewValidatingWebhook(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	kubeClient := fake.NewSimpleClientset()
	certManager := certificate.NewMockManager(mockCtrl)
	stop := make(<-chan struct{})

	testCases := []struct {
		testName    string
		webhookName string
		expErr      string
		mockCall    interface{}
	}{
		{
			testName:    "Error in updateValidatingWebhookCABundle",
			webhookName: "-webhook-name-",
			mockCall:    certManager.EXPECT().IssueCertificate(certificate.CommonName(fmt.Sprintf("%s.%s.svc", validatorServiceName, whc.osmNamespace)), constants.XDSCertificateValidityPeriod),
			expErr:      "validatingwebhookconfigurations.admissionregistration.k8s.io \"-webhook-name-\" not found",
		},
		{
			testName:    "Error in IssueCertificate",
			webhookName: "-webhook-name-",
			mockCall:    certManager.EXPECT().IssueCertificate(certificate.CommonName(fmt.Sprintf("%s.%s.svc", validatorServiceName, whc.osmNamespace)), constants.XDSCertificateValidityPeriod).Return(nil, errors.New("error issuing certificate")),
			expErr:      "error issuing certificate",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			res := NewValidatingWebhook(kubeClient, certManager, whc.osmNamespace, tc.webhookName, stop)
			_ = tc.mockCall
			assert.Equal(tc.expErr, res.Error())
		})
	}
}

func TestConfigMapHandler(t *testing.T) {
	assert := tassert.New(t)
	req := httptest.NewRequest("GET", "/a/b/c", strings.NewReader(admissionRequestBody))
	req.Header = map[string][]string{
		"Content-Type": {"application/json"},
	}
	w := httptest.NewRecorder()
	whc.configMapHandler(w, req)
	resp := w.Result()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	expRes := "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"11111111-2222-3333-4444-555555555555\",\"allowed\":true,\"status\":{\"metadata\":{}}}}"
	assert.Equal(http.StatusOK, resp.StatusCode)
	assert.Equal(expRes, string(bodyBytes))
}

func TestGetAdmissionReqResp(t *testing.T) {
	assert := tassert.New(t)

	requestForNamespace, admissionResp := whc.getAdmissionReqResp([]byte(admissionRequestBody))

	expectedAdmissionResponse := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{Kind: "AdmissionReview", APIVersion: "admission.k8s.io/v1"},
		Request:  nil,
		Response: &admissionv1.AdmissionResponse{
			UID:              "11111111-2222-3333-4444-555555555555",
			Allowed:          true,
			Result:           &metav1.Status{},
			Patch:            nil,
			PatchType:        nil,
			AuditAnnotations: nil,
		},
	}
	assert.Equal("default", requestForNamespace)
	assert.Equal(expectedAdmissionResponse, admissionResp)
}

func TestValidateConfigMap(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		testName string
		req      *admissionv1.AdmissionRequest
		expRes   *admissionv1.AdmissionResponse
	}{
		{
			testName: "Admission request is nil",
			req:      nil,
			expRes: &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Message: errors.New("nil admission request").Error(),
				},
			},
		},
		{
			testName: "Error decoding request",
			req: &admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{
					Raw: []byte("asdf"),
				},
			},
			expRes: &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Message: "couldn't get version/kind; json parse error: json: cannot unmarshal string into Go value of type struct { APIVersion string \"json:\\\"apiVersion,omitempty\\\"\"; Kind string \"json:\\\"kind,omitempty\\\"\" }",
				},
			},
		},
		{
			testName: "Allow updates to configmaps that are not osm-config",
			req: &admissionv1.AdmissionRequest{
				UID: "1234",
				Kind: metav1.GroupVersionKind{
					Version: "/v1",
					Kind:    "ConfigMap",
				},
				Name: "notOsmConfig",
			},
			expRes: &admissionv1.AdmissionResponse{
				Allowed: true,
				Result:  &metav1.Status{Reason: ""},
				UID:     "1234",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			res := whc.validateConfigMap(tc.req)
			assert.Equal(tc.expRes, res)
		})
	}
}

func TestCheckDefaultFields(t *testing.T) {
	assert := tassert.New(t)
	resp := &admissionv1.AdmissionResponse{
		Allowed: true,
		Result:  &metav1.Status{Reason: ""},
	}

	testCases := []struct {
		testName         string
		configMap        corev1.ConfigMap
		expectedResponse *admissionv1.AdmissionResponse
	}{
		{
			testName: "Contains all default fields",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"egress":                           "",
					"enable_debug_server":              "",
					"permissive_traffic_policy_mode":   "",
					"prometheus_scraping":              "",
					"tracing_enable":                   "",
					"use_https_ingress":                "",
					"envoy_log_level":                  "",
					"service_cert_validity_duration":   "",
					"tracing_address":                  "",
					"tracing_port":                     "",
					"tracing_endpoint":                 "",
					"enable_privileged_init_container": "",
					"max_data_plane_connections":       "",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: true,
				Result:  &metav1.Status{Reason: ""},
			},
		},
		{
			testName: "Does not have all default fields",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"egress":                         "",
					"enable_debug_server":            "",
					"permissive_traffic_policy_mode": "",
					"prometheus_scraping":            "",
					"use_https_ingress":              "",
					"envoy_log_level":                "",
					"service_cert_validity_duration": "",
					"tracing_enable":                 "",
					"max_data_plane_connections":     "",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\nenable_privileged_init_container" + doesNotContainDef},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			res := checkDefaultFields(tc.configMap, resp)
			assert.Equal(tc.expectedResponse, res)
		})
	}
}

func TestValidateFields(t *testing.T) {
	assert := tassert.New(t)

	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: whc.osmNamespace,
			Name:      constants.OSMConfigMap,
			Annotations: map[string]string{
				"apple": "banana",
			},
		},
	}
	_, err := whc.kubeClient.CoreV1().ConfigMaps(whc.osmNamespace).Create(context.TODO(), &cm, metav1.CreateOptions{})
	assert.Nil(err)

	testCases := []struct {
		testName         string
		configMap        corev1.ConfigMap
		expectedResponse *admissionv1.AdmissionResponse
	}{
		{
			testName: "Accept valid configMap update",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"egress":                           "true",
					"envoy_log_level":                  "debug",
					"service_cert_validity_duration":   "24h",
					"tracing_port":                     "9411",
					"outbound_ip_range_exclusion_list": "1.1.1.1/32, 2.2.2.2/24",
					"max_data_plane_connections":       "1000",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: true,
				Result:  &metav1.Status{Reason: ""},
			},
		},
		{
			testName: "Reject change to metadata",
			configMap: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"apple": "apple",
					},
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\napple" + cannotChangeMetadata},
			},
		},
		{
			testName: "Reject invalid boolean field update",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"egress": "truie",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\negress" + mustBeBool},
			},
		},
		{
			testName: "Reject invalid envoy_log_level update",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"envoy_log_level": "envoy",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\nenvoy_log_level" + mustBeValidLogLvl},
			},
		},
		{
			testName: "Reject invalid tracing_port update",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"tracing_port": "123456",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\ntracing_port" + mustBeInPortRange},
			},
		},
		{
			testName: "Reject invalid service_cert_validity_duration update",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"service_cert_validity_duration": "1hw",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\nservice_cert_validity_duration" + mustBeValidTime},
			},
		},
		{
			testName: "Reject invalid tracing_port update",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"tracing_port": "1.00",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\ntracing_port" + mustBeInt},
			},
		},
		{
			testName: "Reject configmap with invalid syntax for IP range",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"outbound_ip_range_exclusion_list": "1.1.1.1", // invalid syntax, must be 1.1.1.1/32
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\noutbound_ip_range_exclusion_list" + mustBeValidIPRange},
			},
		},
		{
			testName: "Reject configmap with invalid outbound IP range exclusions",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"outbound_ip_range_exclusion_list": "foobar",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\noutbound_ip_range_exclusion_list" + mustBeValidIPRange},
			},
		},
		{
			testName: "Reject invalid max_data_plane_connections update",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"max_data_plane_connections": "foobar",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\nmax_data_plane_connections" + mustBePositiveInt},
			},
		},
		{
			testName: "Reject invalid max_data_plane_connections update",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"max_data_plane_connections": "-1",
				},
			},
			expectedResponse: &admissionv1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\nmax_data_plane_connections" + mustBePositiveInt},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			resp := &admissionv1.AdmissionResponse{
				Allowed: true,
				Result:  &metav1.Status{Reason: ""},
			}
			res := whc.validateFields(tc.configMap, resp)
			assert.Equal(tc.expectedResponse, res)
		})
	}
}

func TestCheckEnvoyLogLevels(t *testing.T) {
	assert := tassert.New(t)
	tests := map[string]bool{
		"debug":      true,
		"invalidLvl": false,
	}

	for lvl, expRes := range tests {
		res := checkEnvoyLogLevels(fakeField, lvl)
		assert.Equal(expRes, res)
	}
}

func TestCheckBoolFields(t *testing.T) {
	assert := tassert.New(t)
	fakeFields := []string{"field"}
	tests := map[string]bool{
		"true":  true,
		"false": true,
		"f":     true,
		"t":     true,
		"asdf":  false,
		"tre":   false,
	}

	for val, expRes := range tests {
		res := checkBoolFields(fakeField, val, fakeFields)
		assert.Equal(expRes, res)
	}
}

func TestGetPartialValidatingWebhookConfiguration(t *testing.T) {
	assert := tassert.New(t)
	cert := mockCertificate{}
	webhookConfigName := "-webhook-config-name-"
	res := getPartialValidatingWebhookConfiguration(ValidatingWebhookName, cert, webhookConfigName)

	expectedRes := admissionregv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookConfigName,
		},
		Webhooks: []admissionregv1.ValidatingWebhook{
			{
				Name: ValidatingWebhookName,
				ClientConfig: admissionregv1.WebhookClientConfig{
					CABundle: cert.GetCertificateChain(),
				},
				SideEffects: func() *admissionregv1.SideEffectClass {
					sideEffect := admissionregv1.SideEffectClassNone
					return &sideEffect
				}(),
				AdmissionReviewVersions: []string{"v1"},
			},
		},
	}
	assert.Equal(expectedRes, res)
}

func TestUpdateValidatingWebhookCABundle(t *testing.T) {
	assert := tassert.New(t)
	cert := mockCertificate{}
	webhookName := "--webhookName--"
	testWebhookServiceNamespace := "test-namespace"
	testWebhookServiceName := "test-service-name"
	testWebhookServicePath := "/path"
	kubeClient := fake.NewSimpleClientset(&admissionregv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
		Webhooks: []admissionregv1.ValidatingWebhook{
			{
				Name: ValidatingWebhookName,
				ClientConfig: admissionregv1.WebhookClientConfig{
					Service: &admissionregv1.ServiceReference{
						Namespace: testWebhookServiceNamespace,
						Name:      testWebhookServiceName,
						Path:      &testWebhookServicePath,
					},
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"some-key": "some-value",
					},
				},
			},
		},
	})
	err := updateValidatingWebhookCABundle(cert, webhookName, kubeClient)
	assert.Nil(err)
}

type mockCertificate struct{}

func (mc mockCertificate) GetCommonName() certificate.CommonName     { return "" }
func (mc mockCertificate) GetCertificateChain() []byte               { return []byte("chain") }
func (mc mockCertificate) GetPrivateKey() []byte                     { return []byte("key") }
func (mc mockCertificate) GetIssuingCA() []byte                      { return []byte("ca") }
func (mc mockCertificate) GetExpiration() time.Time                  { return time.Now() }
func (mc mockCertificate) GetSerialNumber() certificate.SerialNumber { return "serial_number" }
