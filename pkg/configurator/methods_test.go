package configurator

import (
	"context"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	testclient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

func TestGetMeshConfigCacheKey(t *testing.T) {
	c := CRDClient{
		meshConfigName: "configName",
		osmNamespace:   "namespaceName",
	}
	expected := "namespaceName/configName"
	actual := c.getMeshConfigCacheKey()
	tassert.Equal(t, expected, actual)
}

func TestCreateUpdateConfig(t *testing.T) {
	t.Run("MeshConfig doesn't exist", func(t *testing.T) {
		meshConfigClientSet := testclient.NewSimpleClientset()

		stop := make(chan struct{})
		cfg := newConfiguratorWithCRDClient(meshConfigClientSet, stop, osmNamespace, osmMeshConfigName)
		tassert.Equal(t, &osmConfig{}, cfg.getMeshConfig())
	})

	tests := []struct {
		name                  string
		initialMeshConfigData *v1alpha1.MeshConfigSpec
		checkCreate           func(*tassert.Assertions, Configurator)
		updatedMeshConfigData *v1alpha1.MeshConfigSpec
		checkUpdate           func(*tassert.Assertions, Configurator)
	}{
		{
			name: "default",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					EnablePrivilegedInitContainer: true,
					LogLevel:                      "error",
					MaxDataPlaneConnections:       0,
					ConfigResyncInterval:          "2m",
					EnvoyImage:                    "envoyproxy/envoy-alpine:v0.0.0",
				},
				Traffic: v1alpha1.TrafficSpec{
					EnablePermissiveTrafficPolicyMode: false,
					EnableEgress:                      true,
					UseHTTPSIngress:                   true,
				},
				Observability: v1alpha1.ObservabilitySpec{
					EnableDebugServer:  true,
					PrometheusScraping: true,
					Tracing: v1alpha1.TracingSpec{
						Enable: true,
					},
				},
				Certificate: v1alpha1.CertificateSpec{
					ServiceCertValidityDuration: "24h",
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				expectedConfig := &osmConfig{
					PermissiveTrafficPolicyMode:   false,
					Egress:                        true,
					EnableDebugServer:             true,
					PrometheusScraping:            true,
					TracingEnable:                 true,
					UseHTTPSIngress:               true,
					EnablePrivilegedInitContainer: true,
					EnvoyLogLevel:                 "error",
					EnvoyImage:                    "envoyproxy/envoy-alpine:v0.0.0",
					ServiceCertValidityDuration:   "24h",
					ConfigResyncInterval:          "2m",
					MaxDataPlaneConnections:       0,
				}
				expectedConfigBytes, err := marshalConfigToJSON(expectedConfig)
				assert.Nil(err)

				configBytes, err := cfg.GetMeshConfigJSON()
				assert.Nil(err)
				assert.Equal(string(expectedConfigBytes), string(configBytes))
			},
		},
		{
			name: "IsPermissiveTrafficPolicyMode",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{
				Traffic: v1alpha1.TrafficSpec{
					EnablePermissiveTrafficPolicyMode: true,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsPermissiveTrafficPolicyMode())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Traffic: v1alpha1.TrafficSpec{
					EnablePermissiveTrafficPolicyMode: false,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsPermissiveTrafficPolicyMode())
			},
		},
		{
			name: "IsEgressEnabled",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{
				Traffic: v1alpha1.TrafficSpec{
					EnableEgress: true,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsEgressEnabled())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Traffic: v1alpha1.TrafficSpec{
					EnableEgress: false,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsEgressEnabled())
			},
		},
		{
			name: "IsDebugServerEnabled",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{
				Observability: v1alpha1.ObservabilitySpec{
					EnableDebugServer: true,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsDebugServerEnabled())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Observability: v1alpha1.ObservabilitySpec{
					EnableDebugServer: false,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsDebugServerEnabled())
			},
		},
		{
			name: "IsPrometheusScrapingEnabled",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{
				Observability: v1alpha1.ObservabilitySpec{
					PrometheusScraping: true,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsPrometheusScrapingEnabled())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Observability: v1alpha1.ObservabilitySpec{
					PrometheusScraping: false,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsPrometheusScrapingEnabled())
			},
		},
		{
			name: "IsTracingEnabled",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{
				Observability: v1alpha1.ObservabilitySpec{
					Tracing: v1alpha1.TracingSpec{
						Enable:   true,
						Address:  "myjaeger",
						Port:     12121,
						Endpoint: "/my/endpoint",
					},
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsTracingEnabled())
				assert.Equal("myjaeger", cfg.GetTracingHost())
				assert.Equal(uint32(12121), cfg.GetTracingPort())
				assert.Equal("/my/endpoint", cfg.GetTracingEndpoint())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Observability: v1alpha1.ObservabilitySpec{
					Tracing: v1alpha1.TracingSpec{
						Enable:   false,
						Address:  "myjaeger",
						Port:     12121,
						Endpoint: "/my/endpoint",
					},
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsTracingEnabled())
				assert.Equal(constants.DefaultTracingHost+".-test-osm-namespace-.svc.cluster.local", cfg.GetTracingHost())
				assert.Equal(constants.DefaultTracingPort, cfg.GetTracingPort())
				assert.Equal(constants.DefaultTracingEndpoint, cfg.GetTracingEndpoint())
			},
		},
		{
			name: "UseHTTPSIngress",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{
				Traffic: v1alpha1.TrafficSpec{
					UseHTTPSIngress: true,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.UseHTTPSIngress())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Traffic: v1alpha1.TrafficSpec{
					UseHTTPSIngress: false,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.UseHTTPSIngress())
			},
		},
		{
			name:                  "GetEnvoyLogLevel",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("error", cfg.GetEnvoyLogLevel())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					LogLevel: "info",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("info", cfg.GetEnvoyLogLevel())
			},
		},
		{
			name:                  "GetEnvoyImage",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("envoyproxy/envoy-alpine:v1.17.2", cfg.GetEnvoyImage())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					EnvoyImage: "envoyproxy/envoy-alpine:v1.17.1",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("envoyproxy/envoy-alpine:v1.17.1", cfg.GetEnvoyImage())
			},
		},
		{
			name: "GetServiceCertValidityDuration",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{
				Certificate: v1alpha1.CertificateSpec{
					ServiceCertValidityDuration: "24h",
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(24*time.Hour, cfg.GetServiceCertValidityPeriod())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Certificate: v1alpha1.CertificateSpec{
					ServiceCertValidityDuration: "1h",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(1*time.Hour, cfg.GetServiceCertValidityPeriod())
			},
		},
		{
			name:                  "GetOutboundIPRangeExclusionList",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Nil(cfg.GetOutboundIPRangeExclusionList())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Traffic: v1alpha1.TrafficSpec{
					OutboundIPRangeExclusionList: []string{"1.1.1.1/32, 2.2.2.2/24"},
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal([]string{"1.1.1.1/32", "2.2.2.2/24"}, cfg.GetOutboundIPRangeExclusionList())
			},
		},
		{
			name:                  "GetOutboundPortExclusionList",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Nil(cfg.GetOutboundPortExclusionList())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Traffic: v1alpha1.TrafficSpec{
					OutboundPortExclusionList: []string{"7070, 6080"},
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal([]string{"7070", "6080"}, cfg.GetOutboundPortExclusionList())
			},
		},
		{
			name: "IsPrivilegedInitContainer",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					EnablePrivilegedInitContainer: true,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsPrivilegedInitContainer())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					EnablePrivilegedInitContainer: false,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsPrivilegedInitContainer())
			},
		},
		{
			name:                  "GetResyncInterval",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(interval, time.Duration(0))
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					ConfigResyncInterval: "2m",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(time.Duration(2*time.Minute), interval)
			},
		},
		{
			name:                  "NegativeGetResyncInterval",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(interval, time.Duration(0))
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					ConfigResyncInterval: "Non-duration string",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(interval, time.Duration(0))
			},
		},
		{
			name:                  "GetMaxDataplaneConnections",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(0, cfg.GetMaxDataPlaneConnections())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					MaxDataPlaneConnections: 1000,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(1000, cfg.GetMaxDataPlaneConnections())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			meshConfigClientSet := testclient.NewSimpleClientset()

			// Prepare the pubsub channel
			confChannel := events.GetPubSubInstance().Subscribe(
				announcements.MeshConfigAdded,
				announcements.MeshConfigUpdated)
			defer events.GetPubSubInstance().Unsub(confChannel)

			// Create configurator
			stop := make(chan struct{})
			defer close(stop)
			cfg := NewConfiguratorWithCRDClient(meshConfigClientSet, stop, osmNamespace, osmMeshConfigName)

			meshConfig := v1alpha1.MeshConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmMeshConfigName,
				},
				Spec: *test.initialMeshConfigData,
			}

			_, err := meshConfigClientSet.ConfigV1alpha1().MeshConfigs(osmNamespace).Create(context.TODO(), &meshConfig, metav1.CreateOptions{})
			assert.Nil(err)
			log.Info().Msg("Waiting for create announcement")
			<-confChannel

			test.checkCreate(assert, cfg)

			if test.checkUpdate == nil {
				return
			}

			meshConfig.Spec = *test.updatedMeshConfigData
			_, err = meshConfigClientSet.ConfigV1alpha1().MeshConfigs(osmNamespace).Update(context.TODO(), &meshConfig, metav1.UpdateOptions{})
			assert.Nil(err)

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for update announcement")
			<-confChannel

			test.checkUpdate(assert, cfg)
		})
	}
}
