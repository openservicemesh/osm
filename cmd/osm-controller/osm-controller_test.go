package main

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/httpserver"
)

const (
	testOSMNamespace     = "-test-osm-namespace-"
	testOSMConfigMapName = "-test-osm-config-map-"
)

func toggleDebugServer(enable bool, kubeClient *testclient.Clientset) error {
	updatedConfigMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testOSMNamespace,
			Name:      testOSMConfigMapName,
		},
		Data: map[string]string{
			"enable_debug_server": strconv.FormatBool(enable),
		},
	}
	_, err := kubeClient.CoreV1().ConfigMaps(testOSMNamespace).Update(context.TODO(), &updatedConfigMap, metav1.UpdateOptions{})
	return err
}

func TestDebugServer(t *testing.T) {
	assert := assert.New(t)

	// set up a controller
	stop := make(chan struct{})
	kubeClient, _, cfg, err := setupComponents(testOSMNamespace, testOSMConfigMapName, false, stop)
	assert.NoError(err)

	fakeDebugServer := FakeDebugServer{0, 0, nil, false}
	httpserver.RegisterDebugServer(&fakeDebugServer, cfg)

	assert.Equal(0, fakeDebugServer.startCount)
	assert.Equal(0, fakeDebugServer.stopCount)

	// Start server
	err = toggleDebugServer(true, kubeClient)
	assert.NoError(err)

	assert.Eventually(func() bool {
		return fakeDebugServer.running == true
	}, 2*time.Second, 100*time.Millisecond)
	assert.Eventually(func() bool {
		return fakeDebugServer.startCount == 1
	}, 2*time.Second, 100*time.Millisecond)
	assert.Equal(0, fakeDebugServer.stopCount) // No eventually for an non-expected-to-change value

	// Stop it
	err = toggleDebugServer(false, kubeClient)
	assert.NoError(err)

	assert.Eventually(func() bool {
		return fakeDebugServer.running == false
	}, 2*time.Second, 100*time.Millisecond)
	assert.Eventually(func() bool {
		return fakeDebugServer.stopCount == 1
	}, 2*time.Second, 100*time.Millisecond)
	assert.Equal(1, fakeDebugServer.startCount) // No eventually for an non-expected-to-change value
}

func TestCreateCABundleKubernetesSecret(t *testing.T) {
	assert := assert.New(t)

	certManager := tresor.NewFakeCertManager(nil)
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
	running    bool
}

func (f *FakeDebugServer) Stop() error {
	f.stopCount++
	if f.stopErr != nil {
		return errors.Errorf("Debug server error")
	}
	f.running = false
	return nil
}

func (f *FakeDebugServer) Start() {
	f.startCount++
	f.running = true
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
