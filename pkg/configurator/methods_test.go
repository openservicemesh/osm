package configurator

import (
	"context"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	testclient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

func TestGetMeshConfigCacheKey(t *testing.T) {
	c := Client{
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
		cfg := newConfigurator(meshConfigClientSet, stop, osmNamespace, osmMeshConfigName)
		tassert.Equal(t, &v1alpha1.MeshConfig{}, cfg.getMeshConfig())
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
					InitContainerImage:            "openservicemesh/init:v0.0.0",
				},
				Traffic: v1alpha1.TrafficSpec{
					EnablePermissiveTrafficPolicyMode: false,
					EnableEgress:                      true,
					UseHTTPSIngress:                   true,
				},
				Observability: v1alpha1.ObservabilitySpec{
					OSMLogLevel:       constants.DefaultOSMLogLevel,
					EnableDebugServer: true,
					Tracing: v1alpha1.TracingSpec{
						Enable: true,
					},
				},
				Certificate: v1alpha1.CertificateSpec{
					ServiceCertValidityDuration: "24h",
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				expectedConfig := &v1alpha1.MeshConfigSpec{
					Sidecar: v1alpha1.SidecarSpec{
						EnablePrivilegedInitContainer: true,
						LogLevel:                      "error",
						MaxDataPlaneConnections:       0,
						ConfigResyncInterval:          "2m",
						EnvoyImage:                    "envoyproxy/envoy-alpine:v0.0.0",
						InitContainerImage:            "openservicemesh/init:v0.0.0",
					},
					Traffic: v1alpha1.TrafficSpec{
						EnablePermissiveTrafficPolicyMode: false,
						EnableEgress:                      true,
						UseHTTPSIngress:                   true,
					},
					Observability: v1alpha1.ObservabilitySpec{
						OSMLogLevel:       constants.DefaultOSMLogLevel,
						EnableDebugServer: true,
						Tracing: v1alpha1.TracingSpec{
							Enable: true,
						},
					},
					Certificate: v1alpha1.CertificateSpec{
						ServiceCertValidityDuration: "24h",
					},
				}
				expectedConfigJSON, err := marshalConfigToJSON(expectedConfig)
				assert.Nil(err)

				configJSON, err := cfg.GetMeshConfigJSON()
				assert.Nil(err)
				assert.Equal(expectedConfigJSON, configJSON)
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
				assert.Equal("envoyproxy/envoy-alpine:v1.18.3", cfg.GetEnvoyImage())
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
			name:                  "GetInitContainerImage",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("openservicemesh/init:v0.9.1", cfg.GetInitContainerImage())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					InitContainerImage: "openservicemesh/init:v0.9.0",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("openservicemesh/init:v0.9.0", cfg.GetInitContainerImage())
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
					OutboundIPRangeExclusionList: []string{"1.1.1.1/32", "2.2.2.2/24"},
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
					OutboundPortExclusionList: []int{7070, 6080},
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal([]int{7070, 6080}, cfg.GetOutboundPortExclusionList())
			},
		},
		{
			name:                  "GetIboundPortExclusionList",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Nil(cfg.GetInboundPortExclusionList())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Traffic: v1alpha1.TrafficSpec{
					InboundPortExclusionList: []int{7070, 6080},
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal([]int{7070, 6080}, cfg.GetInboundPortExclusionList())
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
		{
			name:                  "GetProxyResources",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				res := cfg.GetProxyResources()
				assert.Equal(0, len(res.Limits))
				assert.Equal(0, len(res.Requests))
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("1"),
							v1.ResourceMemory: resource.MustParse("256M"),
						},
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("2"),
							v1.ResourceMemory: resource.MustParse("512M"),
						},
					},
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				res := cfg.GetProxyResources()
				assert.Equal(resource.MustParse("1"), res.Requests[v1.ResourceCPU])
				assert.Equal(resource.MustParse("256M"), res.Requests[v1.ResourceMemory])
				assert.Equal(resource.MustParse("2"), res.Limits[v1.ResourceCPU])
				assert.Equal(resource.MustParse("512M"), res.Limits[v1.ResourceMemory])
			},
		},
		{
			name:                  "IsWASMStatsEnabled",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(false, cfg.GetFeatureFlags().EnableWASMStats)
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				FeatureFlags: v1alpha1.FeatureFlags{
					EnableWASMStats: true,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(true, cfg.GetFeatureFlags().EnableWASMStats)
			},
		},
		{
			name:                  "IsEgressPolicyEnabled",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(false, cfg.GetFeatureFlags().EnableEgressPolicy)
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				FeatureFlags: v1alpha1.FeatureFlags{
					EnableEgressPolicy: true,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(true, cfg.GetFeatureFlags().EnableEgressPolicy)
			},
		},
		{
			name:                  "IsMulticlusterModeEnabled",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(false, cfg.GetFeatureFlags().EnableMulticlusterMode)
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				FeatureFlags: v1alpha1.FeatureFlags{
					EnableMulticlusterMode: true,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(true, cfg.GetFeatureFlags().EnableMulticlusterMode)
			},
		},
		{
			name: "OSMLogLevel",
			initialMeshConfigData: &v1alpha1.MeshConfigSpec{
				Observability: v1alpha1.ObservabilitySpec{
					OSMLogLevel: constants.DefaultOSMLogLevel,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(constants.DefaultOSMLogLevel, cfg.GetOSMLogLevel())
			},
			updatedMeshConfigData: &v1alpha1.MeshConfigSpec{
				Observability: v1alpha1.ObservabilitySpec{
					OSMLogLevel: "warn",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("warn", cfg.GetOSMLogLevel())
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
			cfg := NewConfigurator(meshConfigClientSet, stop, osmNamespace, osmMeshConfigName)

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
