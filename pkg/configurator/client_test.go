package configurator

import (
	"context"
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/kubernetes"
)

var _ = Describe("Test OSM ConfigMap parsing", func() {
	kubeClient := testclient.NewSimpleClientset()

	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"
	stop := make(<-chan struct{})
	cfg := newConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: osmNamespace,
			Name:      osmConfigMapName,
		},
	}
	if _, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{}); err != nil {
		log.Fatal().Err(err).Msgf("[TEST] Error creating ConfigMap %s/%s/", configMap.Namespace, configMap.Name)
	}
	<-cfg.GetAnnouncementsChannel()

	Context("Ensure we are able to get reasonable defaults from ConfigMap", func() {

		It("Parsed blank in-mesh CIDR", func() {
			Expect(cfg.getConfigMap().Egress).To(BeFalse())
			actual := cfg.getConfigMap().MeshCIDRRanges
			expected := ""
			Expect(actual).To(Equal(expected))
		})

		It("Parsed default in-mesh CIDR", func() {
			cm := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: map[string]string{
					egressKey: "true",
				},
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &cm, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
			<-cfg.GetAnnouncementsChannel()

			Expect(cfg.getConfigMap().Egress).To(BeTrue())

			actual := cfg.getConfigMap().MeshCIDRRanges
			Expect(actual).To(Equal(defaultInMeshCIDR))

			// TODO(draychev): once we have reasonable defaults this will change to something like:
			// expectedCIDR := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
			var expectedCIDR []string
			Expect(cfg.GetMeshCIDRRanges()).To(Equal(expectedCIDR))
		})

		It("Parsed specific in-mesh CIDR", func() {
			cm := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace,
					Name:      osmConfigMapName,
				},
				Data: map[string]string{
					egressKey:         "true",
					meshCIDRRangesKey: ",  8.8.8.8/24   ,  ,  1.1.0.0/8,   8.8.8.8/24   , someIncorrectlyFormattedCIDR ",
				},
			}
			_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &cm, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			event := <-cfg.GetAnnouncementsChannel()
			log.Info().Msgf("ConfigMap Update Event:  %+v", event.(kubernetes.Event).Value.(*v1.ConfigMap).Data)

			Expect(cfg.getConfigMap().Egress).To(BeTrue())

			actual := cfg.getConfigMap().MeshCIDRRanges
			Expect(actual).To(Equal(cm.Data[meshCIDRRangesKey]))

			Expect(cfg.IsEgressEnabled()).To(BeTrue())

			expectedCIDR := []string{"1.1.0.0/8", "8.8.8.8/24"}
			Expect(cfg.GetMeshCIDRRanges()).To(Equal(expectedCIDR))
		})

		It("Tag matches const key for all fields of OSM ConfigMap struct", func() {
			fieldNameTag := map[string]string{
				"PermissiveTrafficPolicyMode": permissiveTrafficPolicyModeKey,
				"Egress":                      egressKey,
				"PrometheusScraping":          prometheusScrapingKey,
				"ZipkinTracing":               zipkinTracingKey,
				"ZipkinAddress":               zipkinAddressKey,
				"ZipkinPort":                  zipkinPortKey,
				"ZipkinEndpoint":              zipkinEndpointKey,
				"MeshCIDRRanges":              meshCIDRRangesKey,
				"UseHTTPSIngress":             useHTTPSIngressKey,
				"EnvoyLogLevel":               envoyLogLevel,
			}
			t := reflect.TypeOf(osmConfig{})

			actualNumberOfFields := t.NumField()
			expectedNumberOfFields := 10
			Expect(actualNumberOfFields).To(
				Equal(expectedNumberOfFields),
				fmt.Sprintf("Fields have been added or removed from the osmConfig struct -- expected %d, actual %d; please correct this unit test", expectedNumberOfFields, actualNumberOfFields))

			for fieldName, expectedTag := range fieldNameTag {
				f, _ := t.FieldByName("PermissiveTrafficPolicyMode")
				actualtag := f.Tag.Get("yaml")
				Expect(actualtag).To(
					Equal(permissiveTrafficPolicyModeKey),
					fmt.Sprintf("Field %s expected to have tag %s; fuond %s instead", fieldName, expectedTag, actualtag))
			}
		})

		It("Test getBoolValueForKey()", func() {
			cm := &v1.ConfigMap{Data: map[string]string{zipkinTracingKey: "true"}}
			Expect(getBoolValueForKey(cm, zipkinTracingKey)).To(BeTrue())
			Expect(getBoolValueForKey(cm, egressKey)).To(BeFalse())
		})

		It("Test getIntValueForKey()", func() {
			cm := &v1.ConfigMap{Data: map[string]string{zipkinPortKey: "12345"}}
			Expect(getIntValueForKey(cm, zipkinPortKey)).To(Equal(12345))

			cm0 := &v1.ConfigMap{Data: map[string]string{}}
			Expect(getIntValueForKey(cm0, egressKey)).To(Equal(0))
		})

		It("Test getStringValueForKey()", func() {
			cm := &v1.ConfigMap{Data: map[string]string{zipkinEndpointKey: "foo"}}
			Expect(getStringValueForKey(cm, zipkinEndpointKey)).To(Equal("foo"))

			cm0 := &v1.ConfigMap{Data: map[string]string{}}
			Expect(getStringValueForKey(cm0, zipkinEndpointKey)).To(Equal(""))
		})
	})
})
