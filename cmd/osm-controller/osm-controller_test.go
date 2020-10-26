package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

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

func TestConfigureDebugServer(t *testing.T) {
	assert := assert.New(t)
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

		go configureDebugServer(debugServer, mockDebugServer, debugServerRunning, cfg, errCh)
		_, err = kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
		assert.Nil(err)

		close(stop)
		time.Sleep(2 * time.Second)
		if cdst.serverStopped {
			//Checks that debug server is closed
			assert.Equal(debugServer.Server.ListenAndServe(), http.ErrServerClosed)
		}

		stop = make(chan struct{})
		cfg = configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
	}
}

func TestCreateCABundleKubernetesSecret(t *testing.T) {
	assert := assert.New(t)

	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, nil)
	secretName := "--secret--name--"
	namespace := "--namespace--"
	k8sClient := testclient.NewSimpleClientset()

	err := createOrUpdateCABundleKubernetesSecret(k8sClient, certManager, namespace, secretName)
	assert.Nil(err)

	actual, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
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
