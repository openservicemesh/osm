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

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"

	tassert "github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		"apiVersion": "admission.k8s.io/v1beta1",
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
	expRes := "{\"response\":{\"uid\":\"11111111-2222-3333-4444-555555555555\",\"allowed\":true,\"status\":{\"metadata\":{}}}}"
	assert.Equal(http.StatusOK, resp.StatusCode)
	assert.Equal(expRes, string(bodyBytes))
}

func TestGetAdmissionReqResp(t *testing.T) {
	assert := tassert.New(t)

	requestForNamespace, admissionResp := whc.getAdmissionReqResp([]byte(admissionRequestBody))

	expectedAdmissionResponse := v1beta1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{Kind: "", APIVersion: ""},
		Request:  nil,
		Response: &v1beta1.AdmissionResponse{
			UID:              "11111111-2222-3333-4444-555555555555",
			Allowed:          true,
			Result:           &v1.Status{},
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
		req      *v1beta1.AdmissionRequest
		expRes   *v1beta1.AdmissionResponse
	}{
		{
			testName: "Admission request is nil",
			req:      nil,
			expRes: &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: errors.New("nil admission request").Error(),
				},
			},
		},
		{
			testName: "Error decoding request",
			req: &v1beta1.AdmissionRequest{
				Object: runtime.RawExtension{
					Raw: []byte("asdf"),
				},
			},
			expRes: &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: "couldn't get version/kind; json parse error: json: cannot unmarshal string into Go value of type struct { APIVersion string \"json:\\\"apiVersion,omitempty\\\"\"; Kind string \"json:\\\"kind,omitempty\\\"\" }",
				},
			},
		},
		{
			testName: "Allow updates to configmaps that are not osm-config",
			req: &v1beta1.AdmissionRequest{
				UID: "1234",
				Kind: metav1.GroupVersionKind{
					Version: "/v1",
					Kind:    "ConfigMap",
				},
				Name: "notOsmConfig",
			},
			expRes: &v1beta1.AdmissionResponse{
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
	resp := &v1beta1.AdmissionResponse{
		Allowed: true,
		Result:  &metav1.Status{Reason: ""},
	}

	testCases := []struct {
		testName  string
		configMap corev1.ConfigMap
		expRes    *v1beta1.AdmissionResponse
	}{
		{
			testName: "Contains all default fields",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"egress":                         "",
					"enable_debug_server":            "",
					"permissive_traffic_policy_mode": "",
					"prometheus_scraping":            "",
					"tracing_enable":                 "",
					"use_https_ingress":              "",
					"envoy_log_level":                "",
					"service_cert_validity_duration": "",
					"tracing_address":                "",
					"tracing_port":                   "",
					"tracing_endpoint":               "",
				},
			},
			expRes: &v1beta1.AdmissionResponse{
				Allowed: true,
				Result:  &metav1.Status{Reason: ""},
			},
		},
		{
			testName: "Does not have all default fields",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"egress":              "",
					"enable_debug_server": "",
				},
			},
			expRes: &v1beta1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: doesNotContainDef},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			res := checkDefaultFields(tc.configMap, resp)
			assert.Contains(tc.expRes.Result.Status, res.Result.Status)
			assert.Equal(tc.expRes.Allowed, res.Allowed)
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
		testName  string
		configMap corev1.ConfigMap
		expRes    *v1beta1.AdmissionResponse
	}{
		{
			testName: "Accept valid configMap update",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"egress":                         "true",
					"envoy_log_level":                "debug",
					"service_cert_validity_duration": "24h",
					"tracing_port":                   "9411"},
			},
			expRes: &v1beta1.AdmissionResponse{
				Allowed: true,
				Result:  &metav1.Status{Reason: ""},
			},
		},
		{
			testName: "Reject invalid configMap update (error in boolean, envoy lvl, address, port fields and metadata)",
			configMap: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"apple": "apple",
					},
				},
				Data: map[string]string{
					"egress":          "truie",
					"envoy_log_level": "envoy",
					"tracing_port":    "123456"},
			},
			expRes: &v1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Reason: mustBeBool + mustBeValidLogLvl + mustBeInPortRange + cannotChangeMetadata,
				},
			},
		},
		{
			testName: "Reject invalid configmap update (error in cert duration and port fields)",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"service_cert_validity_duration": "1hw",
					"tracing_port":                   "1.00"},
			},
			expRes: &v1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Reason: mustBeValidTime + mustbeInt,
				},
			},
		},
		{
			testName: "Accept configmap with valid outbound IP range exclusions",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"outbound_ip_range_exclusion_list": "1.1.1.1/32, 2.2.2.2/24",
				},
			},
			expRes: &v1beta1.AdmissionResponse{
				Allowed: true,
				Result:  &metav1.Status{Reason: ""},
			},
		},
		{
			testName: "Reject configmap with invalid outbound IP range exclusions",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"outbound_ip_range_exclusion_list": "foobar",
				},
			},
			expRes: &v1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Reason: mustBeValidIPRange,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			resp := &v1beta1.AdmissionResponse{
				Allowed: true,
				Result:  &metav1.Status{Reason: ""},
			}
			res := whc.validateFields(tc.configMap, resp)
			assert.Contains(tc.expRes.Result.Status, res.Result.Status)
			assert.Equal(tc.expRes.Allowed, res.Allowed, res.Result.Reason)
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

	expectedRes := admissionv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookConfigName,
		},
		Webhooks: []admissionv1beta1.ValidatingWebhook{
			{
				Name: ValidatingWebhookName,
				ClientConfig: admissionv1beta1.WebhookClientConfig{
					CABundle: cert.GetCertificateChain(),
				},
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
	kubeClient := fake.NewSimpleClientset(&admissionv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
		Webhooks: []admissionv1beta1.ValidatingWebhook{
			{
				Name: ValidatingWebhookName,
				ClientConfig: admissionv1beta1.WebhookClientConfig{
					Service: &admissionv1beta1.ServiceReference{
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
