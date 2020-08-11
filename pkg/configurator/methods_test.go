package configurator

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Test Envoy configuration creation", func() {
	testCIDRRanges := "10.2.0.0/16 10.0.0.0/16"
	testDebugEnvoyLogLevel := "debug"
	defaultConfigMap := map[string]string{
		permissiveTrafficPolicyModeKey: "false",
		egressKey:                      "true",
		meshCIDRRangesKey:              testCIDRRanges,
		prometheusScrapingKey:          "true",
		zipkinTracingKey:               "true",
		envoyLogLevel:                  testDebugEnvoyLogLevel,
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
		osmNamespace := "-test-osm-namespace-"
		osmConfigMapName := "-test-osm-config-map-"
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

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

			<-cfg.GetAnnouncementsChannel()

			expectedConfig := &osmConfig{
				PermissiveTrafficPolicyMode: false,
				Egress:             true,
				PrometheusScraping: true,
				ZipkinTracing:      true,
				MeshCIDRRanges:     testCIDRRanges,
				EnvoyLogLevel:      testDebugEnvoyLogLevel,
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
		osmNamespace := "-test-osm-namespace-"
		osmConfigMapName := "-test-osm-config-map-"
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

		It("correctly identifies that permissive_traffic_policy_mode is enabled", func() {
			Expect(cfg.IsPermissiveTrafficPolicyMode()).To(BeFalse())
			defaultConfigMap[permissiveTrafficPolicyModeKey] = "true"
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
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsPermissiveTrafficPolicyMode()).To(BeTrue())
		})

		It("correctly identifies that permissive_traffic_policy_mode is disabled", func() {
			defaultConfigMap[permissiveTrafficPolicyModeKey] = "false"
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
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsPermissiveTrafficPolicyMode()).To(BeFalse())
		})
	})

	Context("create OSM config for egress", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		osmNamespace := "-test-osm-namespace-"
		osmConfigMapName := "-test-osm-config-map-"
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

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
			<-cfg.GetAnnouncementsChannel()

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
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsEgressEnabled()).To(BeFalse())
		})
	})

	Context("create OSM config for Prometheus scraping", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		osmNamespace := "-test-osm-namespace-"
		osmConfigMapName := "-test-osm-config-map-"
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

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
			<-cfg.GetAnnouncementsChannel()

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
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsPrometheusScrapingEnabled()).To(BeFalse())
		})
	})

	Context("create OSM config for Zipkin tracing", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		osmNamespace := "-test-osm-namespace-"
		osmConfigMapName := "-test-osm-config-map-"
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

		It("correctly identifies that the config is enabled", func() {
			Expect(cfg.IsZipkinTracingEnabled()).To(BeFalse())
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
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsZipkinTracingEnabled()).To(BeTrue())
		})

		It("correctly identifies that the config is disabled", func() {
			defaultConfigMap[zipkinTracingKey] = "false"
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
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.IsZipkinTracingEnabled()).To(BeFalse())
		})
	})

	Context("create OSM config for mesh CIDR ranges", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		osmNamespace := "-test-osm-namespace-"
		osmConfigMapName := "-test-osm-config-map-"
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

		It("correctly retrieves the mesh CIDR ranges", func() {
			Expect(cfg.GetMeshCIDRRanges()).To(BeEmpty())
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
			<-cfg.GetAnnouncementsChannel()

			expectedMeshCIDRRanges := []string{"10.0.0.0/16", "10.2.0.0/16"}
			Expect(cfg.GetMeshCIDRRanges()).To(Equal(expectedMeshCIDRRanges))
		})
	})

	Context("create OSM config for the Envoy proxy log level", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		osmNamespace := "-test-osm-namespace-"
		osmConfigMapName := "-test-osm-config-map-"
		testInfoEnvoyLogLevel := "info"
		testErrorEnvoyLogLevel := "error"
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

		It("correctly identifies that the Envoy log level is debug", func() {
			Expect(cfg.GetEnvoyLogLevel()).To(Equal(testDebugEnvoyLogLevel))
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
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.GetEnvoyLogLevel()).To(Equal(testDebugEnvoyLogLevel))
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
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.GetEnvoyLogLevel()).To(Equal(testInfoEnvoyLogLevel))
		})

		It("correctly identifies that Envoy log level is error", func() {
			defaultConfigMap[envoyLogLevel] = testErrorEnvoyLogLevel
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
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.GetEnvoyLogLevel()).To(Equal(testErrorEnvoyLogLevel))
		})
	})
})
