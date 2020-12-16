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

	gomock "github.com/golang/mock/gomock"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
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

func TestNewWebhookConfig(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	kubeClient := fake.NewSimpleClientset()
	certManager := certificate.NewMockManager(mockCtrl)
	osmNamespace := "-osm-namespace-"
	stop := make(<-chan struct{})

	testCases := []struct {
		webhookName string
		expErr      string
		mockCall    interface{}
	}{
		{
			webhookName: "-webhook-name-",
			mockCall:    certManager.EXPECT().IssueCertificate(certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMControllerName, osmNamespace)), constants.XDSCertificateValidityPeriod),
			expErr:      "validatingwebhookconfigurations.admissionregistration.k8s.io \"-webhook-name-\" not found",
		},
		{
			webhookName: "-webhook-name-",
			mockCall:    certManager.EXPECT().IssueCertificate(certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMControllerName, osmNamespace)), constants.XDSCertificateValidityPeriod).Return(nil, errors.New("error issuing certificate")),
			expErr:      "error issuing certificate",
		},
	}
	for _, test := range testCases {
		res := NewWebhookConfig(kubeClient, certManager, osmNamespace, test.webhookName, stop)
		_ = test.mockCall
		assert.Equal(test.expErr, res.Error())
	}
}

func TestRunValidatingWebhook(t *testing.T) {

}

func TestConfigMapHandler(t *testing.T) {
	assert := assert.New(t)
	whc := &webhookConfig{}
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
	assert := assert.New(t)

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
	assert := assert.New(t)

	testCases := []struct {
		testName string
		req      *v1beta1.AdmissionRequest
		expRes   *v1beta1.AdmissionResponse
	}{
		{
			testName: "nil admission request",
			req:      nil,
			expRes: &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: errors.New("nil Admission Request").Error(),
				},
			},
		},
		{
			testName: "error decoding request",
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
			testName: "allow updates to configmaps that are not osm-config",
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
	for _, test := range testCases {
		res := whc.validateConfigMap(test.req)
		assert.Equal(test.expRes, res)
	}
}

func TestValidate(t *testing.T) {
	assert := assert.New(t)
	resp := &v1beta1.AdmissionResponse{
		Allowed: true,
		Result:  &metav1.Status{Reason: ""},
	}
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: whc.osmNamespace,
			Name:      "osm-config",
			Annotations: map[string]string{
				"apple": "banana",
			},
		},
	}
	whc.kubeClient.CoreV1().ConfigMaps(whc.osmNamespace).Create(context.TODO(), &cm, metav1.CreateOptions{})

	testCases := []struct {
		testName  string
		configMap corev1.ConfigMap
		expRes    *v1beta1.AdmissionResponse
	}{
		{
			testName: "valid configMap update",
			configMap: corev1.ConfigMap{
				Data: map[string]string{
					"egress":                         "true",
					"envoy_log_level":                "debug",
					"service_cert_validity_duration": "24h",
					"tracing_address":                "abc.123.efg",
					"tracing_port":                   "9411"},
			},
			expRes: &v1beta1.AdmissionResponse{
				Allowed: true,
				Result:  &metav1.Status{Reason: ""},
			},
		},
		{
			testName: "invalid configMap update",
			configMap: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"apple": "apple",
					},
				},
				Data: map[string]string{
					"egress":                         "truie",
					"envoy_log_level":                "envoy",
					"service_cert_validity_duration": "8761h",
					"tracing_address":                "abc.123.efg.",
					"tracing_port":                   "123456"},
			},
			expRes: &v1beta1.AdmissionResponse{
				Allowed: false,
				Result:  &metav1.Status{Reason: "\negress: must be a boolean\nenvoy_log_level: invalid log level\nservice_cert_validity_duration: must be max 8760H (1 year)\ntracing_address: invalid hostname syntax\ntracing_port: must be between 0 and 65535\napple: cannot change metadata"},
			},
		},
	}
	for _, test := range testCases {
		res := whc.validate(test.configMap, resp)
		assert.Equal(test.expRes, res)
	}
}

func TestMatchAddrSyntax(t *testing.T) {
	assert := assert.New(t)
	tests := map[string]bool{
		"abc.efg.678":       true,
		"abc2.7A6.123.":     false,
		".234ab.asdf2.1342": false,
		"asde9.*aed.d34d!`": false,
	}

	for addr, expRes := range tests {
		res := matchAddrSyntax(fakeField, addr)
		assert.Equal(expRes, res)
	}
}

func TestCheckEnvoyLogLevels(t *testing.T) {
	assert := assert.New(t)
	tests := map[string]bool{
		"debug":      true,
		"invalidLvl": false,
	}

	for lvl, expRes := range tests {
		res := checkEnvoyLogLevels(fakeField, lvl, ValidEnvoyLogLevels)
		assert.Equal(expRes, res)
	}
}

func TestCheckBoolFields(t *testing.T) {
	assert := assert.New(t)
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
	assert := assert.New(t)
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
	assert := assert.New(t)
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

func (mc mockCertificate) GetCommonName() certificate.CommonName { return "" }
func (mc mockCertificate) GetCertificateChain() []byte           { return []byte("chain") }
func (mc mockCertificate) GetPrivateKey() []byte                 { return []byte("key") }
func (mc mockCertificate) GetIssuingCA() []byte                  { return []byte("ca") }
func (mc mockCertificate) GetExpiration() time.Time              { return time.Now() }
func (mc mockCertificate) GetSerialNumber() string               { return "serial_number" }
