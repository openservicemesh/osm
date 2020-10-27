package main

import (
	"context"
	"net/http"
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
	"github.com/openservicemesh/osm/pkg/httpserver"
)

type FakeDebugServer struct {
	status                     string
	err                        error
	changeEnableConfigValTo    string
	debugServerRunning         bool
	expectedDebugServerRunning bool
}

func (f *FakeDebugServer) Stop() error {
	f.status = "Stopped"
	if f.err != nil {
		return errors.Errorf("Debug server error")
	}
	return nil
}

func (f *FakeDebugServer) Start() {
	f.status = "Started"
}
func TestConfigureDebugServer(t *testing.T) {
	assert := assert.New(t)
	const testPort = 9999

	defaultConfigMap := map[string]string{
		"enabled_debug_server": "true",
	}
	kubeClient := testclient.NewSimpleClientset()
	stop := make(chan struct{})
	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"
	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: osmNamespace,
			Name:      osmConfigMapName,
		},
		Data: defaultConfigMap,
	}
	_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
	assert.Nil(err)

	mockCtrl := gomock.NewController(t)
	mockDebugServer := debugger.NewMockDebugServer(mockCtrl)
	//ds := httpserver.NewDebugHTTPServer(mockDebugServer, testPort)

	errCh := make(chan error)
	fakeDebugServerTests := []FakeDebugServer{
		{"", nil, "false", true, false},
		{"", nil, "true", false, true},
		{"", errors.Errorf("Debug server error"), "false", true, false},
	}

	for _, f := range fakeDebugServerTests {
		defaultConfigMap["enable_debug_server"] = f.changeEnableConfigValTo
		configMap := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: osmNamespace,
				Name:      osmConfigMapName,
			},
			Data: defaultConfigMap,
		}
		go configureDebugServer(f, mockDebugServer, f.debugServerRunning, cfg, errCh)
		_, err = kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
		close(stop)

		if f.expectedDebugServerRunning {
			assert.Equal("Started", f.status)
		} else {
			assert.Equal("Stopped", f.status)
		}
		assert.Equal(f.expectedDebugServerRunning, f.debugServerRunning)
		if f.err != nil {
			errD := <-errCh
			assert.Equal(f.err, errD)
		}

		stop = make(chan struct{})
		cfg = configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
	}
}

func TestConfigureDebugServer2(t *testing.T) {
	assert := assert.New(t)
	t.Skip()
	const testPort = 9999
	const validRoutePath = "/debug/test1"

	defaultConfigMap := map[string]string{
		"enabled_debug_server": "true",
	}
	kubeClient := testclient.NewSimpleClientset()
	stop := make(chan struct{})
	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"
	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: osmNamespace,
			Name:      osmConfigMapName,
		},
		Data: defaultConfigMap,
	}
	_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
	assert.Nil(err)

	mockCtrl := gomock.NewController(t)
	mockDebugServer := debugger.NewMockDebugServer(mockCtrl)
	mockDebugServer.EXPECT().GetHandlers().Return(map[string]http.Handler{
		validRoutePath: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}).AnyTimes()

	type configDebugServerTest struct {
		defaultEnableDebug bool
		enableDebug        string
		serverStopped      bool
	}

	debugServer := httpserver.NewDebugHTTPServer(mockDebugServer, testPort)

	configDebugServerTests := []configDebugServerTest{
		{true, "false", true},  //stop debug server
		{false, "true", false}, //start debug server
	}
	errCh := make(chan error)

	for _, cdst := range configDebugServerTests {
		defaultConfigMap["enable_debug_server"] = cdst.enableDebug
		configMap := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: osmNamespace,
				Name:      osmConfigMapName,
			},
			Data: defaultConfigMap,
		}
		debugServerRunning := cdst.defaultEnableDebug

		//go configureDebugServer(debugServer, mockDebugServer, debugServerRunning, cfg, errCh)
		_, err = kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
		assert.Nil(err)

		close(stop)
		// if cdst.serverStopped {
		// 	//Checks that debug server is closed
		// 	assert.Equal(debugServer.Server.ListenAndServe(), http.ErrServerClosed)
		// }
		stop = make(chan struct{})
		cfg = configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
	}
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

	expected := "-----BEGIN CERTIFICATE-----\nMIIF"
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
