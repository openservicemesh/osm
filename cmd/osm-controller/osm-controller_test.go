package main

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"testing"

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

type FakeDebugServer struct {
	stopCount int

	stopErr error
	//wg      sync.WaitGroup // create a wait group, this will allow you to block later

}

func (f *FakeDebugServer) Stop() error {
	f.stopCount++
	if f.stopErr != nil {
		return errors.Errorf("Debug server error")
	}
	return nil
}

func (f *FakeDebugServer) Start() {
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

func TestConfigureDebugServerStart(t *testing.T) {
	assert := assert.New(t)

	// set up a controller
	mockCtrl := gomock.NewController(t)
	stop := make(chan struct{})

	kubeClient, _, cfg, err := setupComponents(testOSMNamespace, testOSMConfigMapName, false, stop)
	assert.Nil(err)

	con := &controller{
		debugServerRunning: false,
		debugComponents:    mockDebugConfig(mockCtrl),
		debugServer:        nil,
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go con.configureDebugServer(cfg, &wg)

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
	assert.Nil(err)
	wg.Wait()

	close(stop)

	assert.True(con.debugServerRunning)
	assert.NotNil(con.debugServer)
}

func TestConfigureDebugServerStop(t *testing.T) {
	assert := assert.New(t)

	// set up a controller
	mockCtrl := gomock.NewController(t)
	stop := make(chan struct{})

	kubeClient, _, cfg, err := setupComponents(testOSMNamespace, testOSMConfigMapName, true, stop)
	assert.Nil(err)
	fakeDebugServer := FakeDebugServer{0, nil}
	con := &controller{
		debugServerRunning: true,
		debugComponents:    mockDebugConfig(mockCtrl),
		debugServer:        &fakeDebugServer,
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go con.configureDebugServer(cfg, &wg)

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
	assert.Nil(err)
	wg.Wait()

	close(stop)

	assert.False(con.debugServerRunning)
	assert.Nil(con.debugServer)
}

func TestConfigureDebugServerErr(t *testing.T) {
	assert := assert.New(t)

	// set up a controller
	mockCtrl := gomock.NewController(t)
	stop := make(chan struct{})

	kubeClient, _, cfg, err := setupComponents(testOSMNamespace, testOSMConfigMapName, true, stop)
	assert.Nil(err)
	fakeDebugServer := FakeDebugServer{0, errors.Errorf("Debug server error")}
	con := &controller{
		debugServerRunning: true,
		debugComponents:    mockDebugConfig(mockCtrl),
		debugServer:        &fakeDebugServer,
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go con.configureDebugServer(cfg, &wg)

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
	assert.Nil(err)
	wg.Wait()

	close(stop)

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
	assert.Nil(err)

	actual, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), testName, metav1.GetOptions{})
	assert.Nil(err)

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
