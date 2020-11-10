package main

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/debugger"
)

const (
	validRoutePath       = "/debug/test1"
	testOSMNamespace     = "-test-osm-namespace-"
	testOSMConfigMapName = "-test-osm-config-map-"
)

func TestConfigureDebugServerStart(t *testing.T) {
	assert := assert.New(t)

	// set up a controller
	mockCtrl := gomock.NewController(t)
	stop := make(chan struct{})

	kubeClient, _, cfg, err := setupComponents(testOSMNamespace, testOSMConfigMapName, false, stop)
	if err != nil {
		t.Fatal(err)
	}

	fakeDebugServer := FakeDebugServer{0, 0, nil}
	con := &controller{
		debugServerRunning: false,
		debugComponents:    mockDebugConfig(mockCtrl),
		debugServer:        &fakeDebugServer,
	}

	errs := make(chan error)
	go con.configureDebugServer(cfg, errs)

	updatedConfigMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testOSMNamespace,
			Name:      testOSMConfigMapName,
		},
		Data: map[string]string{
			"enable_debug_server": "true",
		},
	}
	_, err = kubeClient.CoreV1().ConfigMaps(testOSMNamespace).Update(context.TODO(), &updatedConfigMap, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	getErrorOrTimeout(assert, errs, 1*time.Second)
	close(stop)
	assert.Equal(1, fakeDebugServer.startCount)
	assert.Equal(0, fakeDebugServer.stopCount)
	assert.True(con.debugServerRunning)
	assert.NotNil(con.debugServer)
}

func TestConfigureDebugServerStop(t *testing.T) {
	assert := assert.New(t)

	// set up a controller
	mockCtrl := gomock.NewController(t)
	stop := make(chan struct{})

	kubeClient, _, cfg, err := setupComponents(testOSMNamespace, testOSMConfigMapName, true, stop)
	if err != nil {
		t.Fatal(err)
	}

	fakeDebugServer := FakeDebugServer{0, 0, nil}
	con := &controller{
		debugServerRunning: true,
		debugComponents:    mockDebugConfig(mockCtrl),
		debugServer:        &fakeDebugServer,
	}

	errs := make(chan error)
	go con.configureDebugServer(cfg, errs)

	updatedConfigMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testOSMNamespace,
			Name:      testOSMConfigMapName,
		},
		Data: map[string]string{
			"enable_debug_server": "false",
		},
	}
	_, err = kubeClient.CoreV1().ConfigMaps(testOSMNamespace).Update(context.TODO(), &updatedConfigMap, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	getErrorOrTimeout(assert, errs, 1*time.Second)
	close(stop)
	assert.Equal(0, fakeDebugServer.startCount)
	assert.Equal(1, fakeDebugServer.stopCount)
	assert.False(con.debugServerRunning)
	assert.Nil(con.debugServer)
}

func TestConfigureDebugServerErr(t *testing.T) {
	assert := assert.New(t)

	// set up a controller
	mockCtrl := gomock.NewController(t)
	stop := make(chan struct{})

	kubeClient, _, cfg, err := setupComponents(testOSMNamespace, testOSMConfigMapName, true, stop)
	if err != nil {
		t.Fatal(err)
	}

	fakeDebugServer := FakeDebugServer{0, 0, errors.Errorf("Debug server error")}
	con := &controller{
		debugServerRunning: true,
		debugComponents:    mockDebugConfig(mockCtrl),
		debugServer:        &fakeDebugServer,
	}
	errs := make(chan error)
	go con.configureDebugServer(cfg, errs)

	updatedConfigMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testOSMNamespace,
			Name:      testOSMConfigMapName,
		},
		Data: map[string]string{
			"enable_debug_server": "false",
		},
	}
	_, err = kubeClient.CoreV1().ConfigMaps(testOSMNamespace).Update(context.TODO(), &updatedConfigMap, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	getErrorOrTimeout(assert, errs, 1*time.Second)
	close(stop)
	assert.Equal(0, fakeDebugServer.startCount)
	assert.Equal(1, fakeDebugServer.stopCount)
	assert.False(con.debugServerRunning)
	assert.NotNil(con.debugServer)
}

func TestCreateCABundleKubernetesSecret(t *testing.T) {
	assert := assert.New(t)

	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, nil)
	testName := "--secret--name--"
	namespace := "--namespace--"
	k8sClient := testclient.NewSimpleClientset()

	err := createOrUpdateCABundleKubernetesSecret(k8sClient, certManager, namespace, testName)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), testName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	expected := "-----BEGIN CERTIFICATE-----\nMIID"
	stringPEM := string(actual.Data[constants.KubernetesOpaqueSecretCAKey])[:len(expected)]
	assert.Equal(stringPEM, expected)

	expectedRootCert, err := certManager.GetRootCertificate()
	assert.Nil(err)
	assert.Equal(actual.Data[constants.KubernetesOpaqueSecretCAKey], expectedRootCert.GetCertificateChain())
}

func TestJoinURL(t *testing.T) {
	assert := assert.New(t)
	type joinURLtest struct {
		baseURL        string
		path           string
		expectedOutput string
	}
	joinURLtests := []joinURLtest{
		{"http://foo", "/bar", "http://foo/bar"},
		{"http://foo/", "/bar", "http://foo/bar"},
		{"http://foo/", "bar", "http://foo/bar"},
		{"http://foo", "bar", "http://foo/bar"},
	}

	for _, ju := range joinURLtests {
		result := joinURL(ju.baseURL, ju.path)
		assert.Equal(result, ju.expectedOutput)
	}
}

type FakeDebugServer struct {
	stopCount  int
	startCount int
	stopErr    error
}

func (f *FakeDebugServer) Stop() error {
	f.stopCount++
	if f.stopErr != nil {
		return errors.Errorf("Debug server error")
	}
	return nil
}

func (f *FakeDebugServer) Start() {
	f.startCount++
}

func mockDebugConfig(mockCtrl *gomock.Controller) *debugger.MockDebugServer {
	mockDebugConfig := debugger.NewMockDebugServer(mockCtrl)
	mockDebugConfig.EXPECT().GetHandlers().Return(map[string]http.Handler{
		validRoutePath: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}).AnyTimes()
	return mockDebugConfig
}

func setupComponents(namespace, configMapName string, initialDebugServerEnabled bool, stop chan struct{}) (*testclient.Clientset, v1.ConfigMap, configurator.Configurator, error) {
	kubeClient := testclient.NewSimpleClientset()

	defaultConfigMap := map[string]string{
		"enable_debug_server": strconv.FormatBool(initialDebugServerEnabled),
	}
	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      configMapName,
		},
		Data: defaultConfigMap,
	}
	_, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
	if err != nil {
		return kubeClient, configMap, nil, err
	}
	cfg := configurator.NewConfigurator(kubeClient, stop, namespace, configMapName)
	return kubeClient, configMap, cfg, nil
}

func getErrorOrTimeout(assert *assert.Assertions, errs <-chan error, timeout time.Duration) {
	select {
	case <-errs:
	case <-time.After(timeout):
		assert.Fail("failed to receive error after " + timeout.String())
	}
}
