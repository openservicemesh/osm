package configurator

import (
	"context"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

func TestGetConfigMapCacheKey(t *testing.T) {
	c := Client{
		osmConfigMapName: "mapName",
		osmNamespace:     "namespaceName",
	}
	expected := "namespaceName/mapName"
	actual := c.getConfigMapCacheKey()
	tassert.Equal(t, expected, actual)
}

func TestCreateUpdateConfig(t *testing.T) {
	t.Run("ConfigMap doesn't exist", func(t *testing.T) {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		tassert.Equal(t, &osmConfig{}, cfg.(*Client).getConfigMap())
	})

	tests := []struct {
		name                 string
		initialConfigMapData map[string]string
		checkCreate          func(*tassert.Assertions, Configurator)
		updatedConfigMapData map[string]string
		checkUpdate          func(*tassert.Assertions, Configurator)
	}{
		{
			name: "default",
			initialConfigMapData: map[string]string{
				PermissiveTrafficPolicyModeKey: "false",
				egressKey:                      "true",
				enableDebugServer:              "true",
				prometheusScrapingKey:          "true",
				tracingEnableKey:               "true",
				useHTTPSIngressKey:             "true",
				enablePrivilegedInitContainer:  "true",
				envoyLogLevel:                  "error",
				serviceCertValidityDurationKey: "24h",
				configResyncInterval:           "2m",
				maxDataPlaneConnectionsKey:     "0",
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
					ServiceCertValidityDuration:   "24h",
					ConfigResyncInterval:          "2m",
					MaxDataPlaneConnections:       0,
				}
				expectedConfigBytes, err := marshalConfigToJSON(expectedConfig)
				assert.Nil(err)

				configBytes, err := cfg.GetConfigMap()
				assert.Nil(err)
				assert.Equal(string(expectedConfigBytes), string(configBytes))
			},
		},
		{
			name: "IsPermissiveTrafficPolicyMode",
			initialConfigMapData: map[string]string{
				PermissiveTrafficPolicyModeKey: "true",
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsPermissiveTrafficPolicyMode())
			},
			updatedConfigMapData: map[string]string{
				PermissiveTrafficPolicyModeKey: "false",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsPermissiveTrafficPolicyMode())
			},
		},
		{
			name: "IsEgressEnabled",
			initialConfigMapData: map[string]string{
				egressKey: "true",
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsEgressEnabled())
			},
			updatedConfigMapData: map[string]string{
				egressKey: "false",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsEgressEnabled())
			},
		},
		{
			name: "IsDebugServerEnabled",
			initialConfigMapData: map[string]string{
				enableDebugServer: "true",
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsDebugServerEnabled())
			},
			updatedConfigMapData: map[string]string{
				enableDebugServer: "false",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsDebugServerEnabled())
			},
		},
		{
			name: "IsPrometheusScrapingEnabled",
			initialConfigMapData: map[string]string{
				prometheusScrapingKey: "true",
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsPrometheusScrapingEnabled())
			},
			updatedConfigMapData: map[string]string{
				prometheusScrapingKey: "false",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsPrometheusScrapingEnabled())
			},
		},
		{
			name: "IsTracingEnabled",
			initialConfigMapData: map[string]string{
				tracingEnableKey:   "true",
				tracingAddressKey:  "myjaeger",
				tracingPortKey:     "12121",
				tracingEndpointKey: "/my/endpoint",
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsTracingEnabled())
				assert.Equal("myjaeger", cfg.GetTracingHost())
				assert.Equal(uint32(12121), cfg.GetTracingPort())
				assert.Equal("/my/endpoint", cfg.GetTracingEndpoint())
			},
			updatedConfigMapData: map[string]string{
				tracingEnableKey:   "false",
				tracingAddressKey:  "myjaeger",
				tracingPortKey:     "12121",
				tracingEndpointKey: "/my/endpoint",
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
			initialConfigMapData: map[string]string{
				useHTTPSIngressKey: "true",
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.UseHTTPSIngress())
			},
			updatedConfigMapData: map[string]string{
				useHTTPSIngressKey: "false",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.UseHTTPSIngress())
			},
		},
		{
			name:                 "GetEnvoyLogLevel",
			initialConfigMapData: map[string]string{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("error", cfg.GetEnvoyLogLevel())
			},
			updatedConfigMapData: map[string]string{
				envoyLogLevel: "info",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("info", cfg.GetEnvoyLogLevel())
			},
		},
		{
			name: "GetServiceCertValidityDuration",
			initialConfigMapData: map[string]string{
				serviceCertValidityDurationKey: "5", // invalid, should default to 24h
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(24*time.Hour, cfg.GetServiceCertValidityPeriod())
			},
			updatedConfigMapData: map[string]string{
				serviceCertValidityDurationKey: "1h",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(1*time.Hour, cfg.GetServiceCertValidityPeriod())
			},
		},
		{
			name:                 "GetOutboundIPRangeExclusionList",
			initialConfigMapData: map[string]string{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Nil(cfg.GetOutboundIPRangeExclusionList())
			},
			updatedConfigMapData: map[string]string{
				outboundIPRangeExclusionListKey: "1.1.1.1/32, 2.2.2.2/24",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal([]string{"1.1.1.1/32", "2.2.2.2/24"}, cfg.GetOutboundIPRangeExclusionList())
			},
		},
		{
			name: "IsPrivilegedInitContainer",
			initialConfigMapData: map[string]string{
				enablePrivilegedInitContainer: "true",
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsPrivilegedInitContainer())
			},
			updatedConfigMapData: map[string]string{
				enablePrivilegedInitContainer: "false",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsPrivilegedInitContainer())
			},
		},
		{
			name:                 "GetResyncInterval",
			initialConfigMapData: map[string]string{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(interval, time.Duration(0))
			},
			updatedConfigMapData: map[string]string{
				configResyncInterval: "2m",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(time.Duration(2*time.Minute), interval)
			},
		},
		{
			name:                 "NegativeGetResyncInterval",
			initialConfigMapData: map[string]string{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(interval, time.Duration(0))
			},
			updatedConfigMapData: map[string]string{
				configResyncInterval: "Non-duration string",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(interval, time.Duration(0))
			},
		},
		{
			name:                 "GetMaxDataplaneConnections",
			initialConfigMapData: map[string]string{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(0, cfg.GetMaxDataPlaneConnections())
			},
			updatedConfigMapData: map[string]string{
				maxDataPlaneConnectionsKey: "1000",
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(1000, cfg.GetMaxDataPlaneConnections())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			kubeClient := testclient.NewSimpleClientset()

			// Prepare the pubsub channel
			confChannel := events.GetPubSubInstance().Subscribe(announcements.ConfigMapAdded, announcements.ConfigMapUpdated)
			defer events.GetPubSubInstance().Unsub(confChannel)

			// Create configurator
			stop := make(chan struct{})
			defer close(stop)
			cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

			// Issue config map create
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: test.initialConfigMapData,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			assert.Nil(err)

			log.Info().Msg("Waiting for create announcement")
			<-confChannel

			test.checkCreate(assert, cfg)

			if test.checkUpdate == nil {
				return
			}

			configMap.Data = test.updatedConfigMapData
			_, err = kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
			assert.Nil(err)

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for update announcement")
			<-confChannel

			test.checkUpdate(assert, cfg)
		})
	}
}
