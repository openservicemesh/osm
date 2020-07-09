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
	Context("create envoy config", func() {
		kubeClient := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		osmNamespace := "-test-osm-namespace-"
		osmConfigMapName := "-test-osm-config-map-"
		cfg := NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

		It("correctly creates a cache key", func() {
			c := Client{
				osmConfigMapName: "mapName",
				osmNamespace:     "namespaceName",
			}
			expected := "namespaceName/mapName"
			actual := c.getConfigMapCacheKey()
			Expect(actual).To(Equal(expected))
		})

		It("correctly identifies whether the service mesh is in allow_all mode", func() {
			Expect(cfg.IsAllowAll()).To(BeFalse())
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: map[string]string{
					"osm.conf": `
config_version: 111
allow_all: true
`,
				},
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the config map change to propagate to the cache.
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.GetOSMNamespace()).To(Equal(osmNamespace))
			configMapData, err := cfg.GetConfigMap()
			Expect(err).ToNot(HaveOccurred())

			expectedConfigMap := `{
    "ConfigVersion": 111,
    "AllowAll": true
}`

			Expect(string(configMapData)).To(Equal(expectedConfigMap))
			Expect(cfg.IsAllowAll()).To(BeTrue())
		})
	})
})
