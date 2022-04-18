package main

import (
	"context"
	"os"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakeKube "k8s.io/client-go/kubernetes/fake"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
)

var testNamespace = "test-namespace"

var testMeshConfig *configv1alpha3.MeshConfig = &configv1alpha3.MeshConfig{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      meshConfigName,
	},
	Spec: configv1alpha3.MeshConfigSpec{},
}

var testPresetMeshConfigMap *corev1.ConfigMap = &corev1.ConfigMap{
	TypeMeta: metav1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      presetMeshConfigName,
		Namespace: testNamespace,
	},
	Data: map[string]string{
		presetMeshConfigJSONKey: `{
"sidecar": {
	"enablePrivilegedInitContainer": false,
	"logLevel": "error",
	"maxDataPlaneConnections": 0,
	"envoyImage": "envoyproxy/envoy-alpine:v1.19.3@sha256:874e699857e023d9234b10ffc5af39ccfc9011feab89638e56ac4042ecd4b0f3",
	"initContainerImage": "openservicemesh/init:latest-main",
	"configResyncInterval": "2s"
},
"traffic": {
	"enableEgress": true,
	"useHTTPSIngress": false,
	"enablePermissiveTrafficPolicyMode": true,
	"outboundPortExclusionList": [],
	"inboundPortExclusionList": [],
	"outboundIPRangeExclusionList": []
},
"observability": {
	"enableDebugServer": false,
	"osmLogLevel": "trace",
	"tracing": {
	 "enable": false
	}
},
"certificate": {
	"serviceCertValidityDuration": "23h"
},
"featureFlags": {
	"enableWASMStats": false,
	"enableEgressPolicy": true,
	"enableMulticlusterMode": false,
	"enableAsyncProxyServiceMapping": false,
	"enableIngressBackendPolicy": true,
	"enableEnvoyActiveHealthChecks": true,
	"enableSnapshotCacheMode": true,
	"enableRetryPolicy": false
	}
}`,
	},
}

func TestBuildDefaultMeshConfig(t *testing.T) {
	assert := tassert.New(t)

	meshConfig := buildDefaultMeshConfig(testPresetMeshConfigMap)
	assert.Equal(meshConfig.Name, meshConfigName)
	assert.Equal(meshConfig.Spec.Sidecar.LogLevel, "error")
	assert.Equal(meshConfig.Spec.Sidecar.ConfigResyncInterval, "2s")
	assert.False(meshConfig.Spec.Sidecar.EnablePrivilegedInitContainer)
	assert.True(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode)
	assert.True(meshConfig.Spec.Traffic.EnableEgress)
	assert.False(meshConfig.Spec.FeatureFlags.EnableWASMStats)
	assert.False(meshConfig.Spec.Observability.EnableDebugServer)
	assert.Equal(meshConfig.Spec.Certificate.ServiceCertValidityDuration, "23h")
	assert.True(meshConfig.Spec.FeatureFlags.EnableIngressBackendPolicy)
	assert.True(meshConfig.Spec.FeatureFlags.EnableEnvoyActiveHealthChecks)
	assert.False(meshConfig.Spec.FeatureFlags.EnableRetryPolicy)
}

func TestValidateCLIParams(t *testing.T) {
	assert := tassert.New(t)

	// save original global values
	prevOsmNamespace := osmNamespace

	tests := []struct {
		caseName string
		setup    func()
		verify   func(error)
	}{
		{
			caseName: "osm-namespace is empty",
			setup: func() {
				osmNamespace = ""
			},
			verify: func(err error) {
				assert.NotNil(err)
				assert.Contains(err.Error(), "--osm-namespace")
			},
		},
		{
			caseName: "osm-namespace is valid",
			setup: func() {
				osmNamespace = "valid-ns"
			},
			verify: func(err error) {
				assert.NotNil(err)
				assert.Contains(err.Error(), "--ca-bundle-secret-name")
			},
		},
		{
			caseName: "osm-namespace and ca-bundle-secret-name is valid",
			setup: func() {
				osmNamespace = "valid-ns"
				caBundleSecretName = "valid-ca-bundle"
			},
			verify: func(err error) {
				assert.Nil(err)
			},
		},
	}

	for _, tc := range tests {
		tc.setup()
		err := validateCLIParams()
		tc.verify(err)
	}

	// restore original global values
	osmNamespace = prevOsmNamespace
}

func TestCreateDefaultMeshConfig(t *testing.T) {
	assert := tassert.New(t)

	tests := []struct {
		name                    string
		namespace               string
		kubeClient              kubernetes.Interface
		meshConfigClient        configClientset.Interface
		expectDefaultMeshConfig bool
		expectErr               bool
	}{
		{
			name:                    "successfully create default meshconfig from preset configmap",
			namespace:               testNamespace,
			kubeClient:              fakeKube.NewSimpleClientset([]runtime.Object{testPresetMeshConfigMap}...),
			meshConfigClient:        fakeConfig.NewSimpleClientset(),
			expectDefaultMeshConfig: true,
			expectErr:               false,
		},
		{
			name:                    "preset configmap does not exist",
			namespace:               testNamespace,
			kubeClient:              fakeKube.NewSimpleClientset(),
			meshConfigClient:        fakeConfig.NewSimpleClientset(),
			expectDefaultMeshConfig: false,
			expectErr:               true,
		},
		{
			name:                    "default MeshConfig already exists",
			namespace:               testNamespace,
			kubeClient:              fakeKube.NewSimpleClientset([]runtime.Object{testPresetMeshConfigMap}...),
			meshConfigClient:        fakeConfig.NewSimpleClientset([]runtime.Object{testMeshConfig}...),
			expectDefaultMeshConfig: true,
			expectErr:               false,
		},
	}

	for _, tc := range tests {
		b := bootstrap{
			kubeClient:       tc.kubeClient,
			meshConfigClient: tc.meshConfigClient,
			namespace:        tc.namespace,
		}

		err := b.createDefaultMeshConfig()
		if tc.expectErr {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}

		_, err = b.meshConfigClient.ConfigV1alpha2().MeshConfigs(b.namespace).Get(context.TODO(), meshConfigName, metav1.GetOptions{})
		if tc.expectDefaultMeshConfig {
			if err == nil {
				assert.Nil(err)
			}
		} else {
			if err == nil {
				assert.NotNil(err)
			}
		}
	}
}

func TestEnsureMeshConfig(t *testing.T) {
	assert := tassert.New(t)

	tests := []struct {
		name             string
		namespace        string
		kubeClient       kubernetes.Interface
		meshConfigClient configClientset.Interface
		expectErr        bool
	}{
		{
			name:             "MeshConfig found",
			namespace:        testNamespace,
			kubeClient:       fakeKube.NewSimpleClientset(),
			meshConfigClient: fakeConfig.NewSimpleClientset([]runtime.Object{testMeshConfig}...),
			expectErr:        false,
		},
		{
			name:             "MeshConfig not found but successfully created",
			namespace:        testNamespace,
			kubeClient:       fakeKube.NewSimpleClientset([]runtime.Object{testPresetMeshConfigMap}...),
			meshConfigClient: fakeConfig.NewSimpleClientset(),
			expectErr:        false,
		},
		{
			name:             "MeshConfig not found and error creating it",
			namespace:        testNamespace,
			kubeClient:       fakeKube.NewSimpleClientset(),
			meshConfigClient: fakeConfig.NewSimpleClientset(),
			expectErr:        true,
		},
	}

	for _, tc := range tests {
		b := bootstrap{
			kubeClient:       tc.kubeClient,
			meshConfigClient: tc.meshConfigClient,
			namespace:        tc.namespace,
		}

		err := b.ensureMeshConfig()
		if tc.expectErr {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
	}
}

func TestGetBootstrapPod(t *testing.T) {
	assert := tassert.New(t)
	testPodName := "test-pod-name"
	testNamespace := "test-namespace"
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespace,
		},
	}

	tests := []struct {
		name                string
		namespace           string
		bootstrapPodNameEnv string
		kubeClient          kubernetes.Interface
		expectErr           bool
	}{
		{
			name:                "BOOTSTRAP_POD_NAME env var not set",
			namespace:           testNamespace,
			bootstrapPodNameEnv: "",
			kubeClient:          fakeKube.NewSimpleClientset(),
			expectErr:           true,
		},
		{
			name:                "BOOTSTRAP_POD_NAME env var set correctly and pod exists",
			namespace:           testNamespace,
			bootstrapPodNameEnv: testPodName,
			kubeClient:          fakeKube.NewSimpleClientset([]runtime.Object{testPod}...),
			expectErr:           false,
		},
		{
			name:                "BOOTSTRAP_POD_NAME env var set incorrectly",
			namespace:           testNamespace,
			bootstrapPodNameEnv: "something-random",
			kubeClient:          fakeKube.NewSimpleClientset([]runtime.Object{testPod}...),
			expectErr:           true,
		},
	}

	for _, tc := range tests {
		b := bootstrap{
			namespace:  tc.namespace,
			kubeClient: tc.kubeClient,
		}
		defer func() {
			err := resetEnv("BOOTSTRAP_POD_NAME", os.Getenv("BOOTSTRAP_POD_NAME"))
			assert.Nil(err)
		}()

		err := os.Setenv("BOOTSTRAP_POD_NAME", tc.bootstrapPodNameEnv)
		assert.Nil(err)

		_, err = b.getBootstrapPod()
		if tc.expectErr {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
	}
}

func resetEnv(key, val string) error {
	return os.Setenv(key, val)
}
