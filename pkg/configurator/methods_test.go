package configurator

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

var _ = Describe("Test Envoy configuration creation", func() {
	testErrorEnvoyLogLevel := "error"
	//noling: goconst
	defaultConfigMap := map[string]string{
		PermissiveTrafficPolicyModeKey: "false",
		egressKey:                      "true",
		enableDebugServer:              "true",
		prometheusScrapingKey:          "true",
		tracingEnableKey:               "true",
		envoyLogLevel:                  testErrorEnvoyLogLevel,
		serviceCertValidityDurationKey: "24h",
	}

	Context("create OSM configurator client", func() {
		It("correctly creates a cache key", func() {
			c := Client{
				osmConfigMapName: "mapName",
				osmNamespace:     "namespaceName",
			}
			expected := "namespaceName/mapName"
			actual := c.getConfigMapCacheKey()
			Expect(actual).To(Equal(expected))
		})
	})

	Context("create OSM config with default values", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		var confChannel chan interface{}

		BeforeEach(func() {
			confChannel = events.GetPubSubInstance().Subscribe(
				announcements.ConfigMapAdded,
				announcements.ConfigMapDeleted,
				announcements.ConfigMapUpdated)
		})

		AfterEach(func() {
			events.GetPubSubInstance().Unsub(confChannel)
		})

		It("test GetConfigMap", func() {
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			<-confChannel

			expectedConfig := &osmConfig{
				PermissiveTrafficPolicyMode: false,
				Egress:                      true,
				EnableDebugServer:           true,
				PrometheusScraping:          true,
				TracingEnable:               true,
				EnvoyLogLevel:               testErrorEnvoyLogLevel,
				ServiceCertValidityDuration: "24h",
			}
			expectedConfigBytes, err := marshalConfigToJSON(expectedConfig)
			Expect(err).ToNot(HaveOccurred())

			configBytes, err := cfg.GetConfigMap()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(configBytes)).To(Equal(string(expectedConfigBytes)))
		})
	})

	Context("create OSM config for permissive_traffic_policy_mode", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		var confChannel chan interface{}

		BeforeEach(func() {
			confChannel = events.GetPubSubInstance().Subscribe(
				announcements.ConfigMapAdded,
				announcements.ConfigMapDeleted,
				announcements.ConfigMapUpdated)
		})

		AfterEach(func() {
			events.GetPubSubInstance().Unsub(confChannel)
		})

		It("correctly identifies that permissive_traffic_policy_mode is enabled", func() {
			Expect(cfg.IsPermissiveTrafficPolicyMode()).To(BeFalse())
			defaultConfigMap[PermissiveTrafficPolicyModeKey] = "true"
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsPermissiveTrafficPolicyMode()).To(BeTrue())
		})

		It("correctly identifies that permissive_traffic_policy_mode is disabled", func() {
			defaultConfigMap[PermissiveTrafficPolicyModeKey] = "false" //nolint: goconst
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsPermissiveTrafficPolicyMode()).To(BeFalse())
		})
	})

	Context("create OSM config for egress", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		var confChannel chan interface{}

		BeforeEach(func() {
			confChannel = events.GetPubSubInstance().Subscribe(
				announcements.ConfigMapAdded,
				announcements.ConfigMapDeleted,
				announcements.ConfigMapUpdated)
		})

		AfterEach(func() {
			events.GetPubSubInstance().Unsub(confChannel)
		})

		It("correctly identifies that egress is enabled", func() {
			Expect(cfg.IsEgressEnabled()).To(BeFalse())
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsEgressEnabled()).To(BeTrue())
		})

		It("correctly identifies that egress is disabled", func() {
			defaultConfigMap[egressKey] = "false"
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsEgressEnabled()).To(BeFalse())
		})
	})

	Context("create OSM config for osm debug HTTP server", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		var confChannel chan interface{}

		BeforeEach(func() {
			confChannel = events.GetPubSubInstance().Subscribe(
				announcements.ConfigMapAdded,
				announcements.ConfigMapDeleted,
				announcements.ConfigMapUpdated)
		})

		AfterEach(func() {
			events.GetPubSubInstance().Unsub(confChannel)
		})

		It("correctly identifies that the debug server is enabled", func() {
			Expect(cfg.IsDebugServerEnabled()).To(BeFalse())
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsDebugServerEnabled()).To(BeTrue())
		})
	})

	Context("create OSM config for Prometheus scraping", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		var confChannel chan interface{}
		BeforeEach(func() {
			confChannel = events.GetPubSubInstance().Subscribe(
				announcements.ConfigMapAdded,
				announcements.ConfigMapDeleted,
				announcements.ConfigMapUpdated)
		})

		AfterEach(func() {
			events.GetPubSubInstance().Unsub(confChannel)
		})

		It("correctly identifies that the config is enabled", func() {
			Expect(cfg.IsPrometheusScrapingEnabled()).To(BeFalse())
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsPrometheusScrapingEnabled()).To(BeTrue())
		})

		It("correctly identifies that the config is disabled", func() {
			defaultConfigMap[prometheusScrapingKey] = "false"
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsPrometheusScrapingEnabled()).To(BeFalse())
		})
	})

	Context("create OSM config for tracing", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		var confChannel chan interface{}

		BeforeEach(func() {
			confChannel = events.GetPubSubInstance().Subscribe(
				announcements.ConfigMapAdded,
				announcements.ConfigMapDeleted,
				announcements.ConfigMapUpdated)
		})

		AfterEach(func() {
			events.GetPubSubInstance().Unsub(confChannel)
		})

		It("correctly identifies that the config is enabled", func() {
			Expect(cfg.IsTracingEnabled()).To(BeFalse())
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsTracingEnabled()).To(BeTrue())
		})

		It("correctly identifies that the config is disabled", func() {
			defaultConfigMap[tracingEnableKey] = "false"
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsTracingEnabled()).To(BeFalse())
		})
	})

	Context("create OSM config for the Envoy proxy log level", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		testInfoEnvoyLogLevel := "info"
		testDebugEnvoyLogLevel := "debug"
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		var confChannel chan interface{}

		BeforeEach(func() {
			confChannel = events.GetPubSubInstance().Subscribe(
				announcements.ConfigMapAdded,
				announcements.ConfigMapDeleted,
				announcements.ConfigMapUpdated)
		})

		AfterEach(func() {
			events.GetPubSubInstance().Unsub(confChannel)
		})

		It("correctly identifies that the Envoy log level is error", func() {
			Expect(cfg.GetEnvoyLogLevel()).To(Equal(testErrorEnvoyLogLevel))
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.GetEnvoyLogLevel()).To(Equal(testErrorEnvoyLogLevel))
		})

		It("correctly identifies that Envoy log level is info", func() {
			defaultConfigMap[envoyLogLevel] = testInfoEnvoyLogLevel
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel
			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.GetEnvoyLogLevel()).To(Equal(testInfoEnvoyLogLevel))
		})

		It("correctly identifies that Envoy log level is debug", func() {
			defaultConfigMap[envoyLogLevel] = testDebugEnvoyLogLevel
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.GetEnvoyLogLevel()).To(Equal(testDebugEnvoyLogLevel))
		})
	})

	Context("create OSM config service cert validity period", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		var confChannel chan interface{}

		BeforeEach(func() {
			confChannel = events.GetPubSubInstance().Subscribe(
				announcements.ConfigMapAdded,
				announcements.ConfigMapDeleted,
				announcements.ConfigMapUpdated)
		})

		AfterEach(func() {
			events.GetPubSubInstance().Unsub(confChannel)
		})

		It("correctly retrieves the default service cert validity duration when an invalid value is specified", func() {
			defaultConfigMap[serviceCertValidityDurationKey] = "5" // no units, so invalid
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			<-confChannel

			Expect(cfg.GetServiceCertValidityPeriod()).To(Equal(time.Duration(24 * time.Hour)))
		})

		It("correctly retrieves the service cert validity duration", func() {
			defaultConfigMap[serviceCertValidityDurationKey] = "1h"
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			<-confChannel

			Expect(cfg.GetServiceCertValidityPeriod()).To(Equal(time.Duration(1 * time.Hour)))
		})
	})

	Context("test outbound_ip_range_exclusion_list", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		var confChannel chan interface{}

		BeforeEach(func() {
			confChannel = events.GetPubSubInstance().Subscribe(
				announcements.ConfigMapAdded,
				announcements.ConfigMapDeleted,
				announcements.ConfigMapUpdated)
		})

		AfterEach(func() {
			events.GetPubSubInstance().Unsub(confChannel)
		})

		It("correctly returns an empty list when no exclusion list is specified", func() {
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			log.Info().Msg("Waiting for announcement")
			<-confChannel

			Expect(cfg.GetOutboundIPRangeExclusionList()).To(BeNil())
		})

		It("correctly retrieves the IP ranges to exclude", func() {
			defaultConfigMap[outboundIPRangeExclusionListKey] = "1.1.1.1/32, 2.2.2.2/24"
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: defaultConfigMap,
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			<-confChannel

			expected := []string{"1.1.1.1/32", "2.2.2.2/24"}
			actual := cfg.GetOutboundIPRangeExclusionList()
			Expect(actual).Should(HaveLen(len(expected)))
			Expect(actual).Should(ConsistOf(expected))
		})
	})
})
