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
	defaultConfigMap := map[string]string{
		permissiveTrafficPolicyModeKey: "false",
		egressKey:                      "true",
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
})
