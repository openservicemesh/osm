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
)

var _ = Describe("Test OSM ConfigMap parsing", func() {
	defer GinkgoRecover()

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
		GinkgoT().Fatalf("[TEST] Error creating ConfigMap %s/%s/: %s", configMap.Namespace, configMap.Name, err.Error())
	}
	<-cfg.GetAnnouncementsChannel()

	Context("Ensure we are able to get reasonable defaults from ConfigMap", func() {

		It("Tag matches const key for all fields of OSM ConfigMap struct", func() {
			fieldNameTag := map[string]string{
				"PermissiveTrafficPolicyMode": permissiveTrafficPolicyModeKey,
				"Egress":                      egressKey,
				"EnableDebugServer":           enableDebugServer,
				"PrometheusScraping":          prometheusScrapingKey,
				"TracingEnable":               tracingEnableKey,
				"TracingAddress":              tracingAddressKey,
				"TracingPort":                 tracingPortKey,
				"TracingEndpoint":             tracingEndpointKey,
				"UseHTTPSIngress":             useHTTPSIngressKey,
				"EnvoyLogLevel":               envoyLogLevel,
				"ServiceCertValidityDuration": serviceCertValidityDurationKey,
			}
			t := reflect.TypeOf(osmConfig{})

			expectedNumberOfFields := t.NumField()
			actualNumberOfFields := len(fieldNameTag)

			Expect(expectedNumberOfFields).To(
				Equal(actualNumberOfFields),
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
			cm := &v1.ConfigMap{Data: map[string]string{tracingEnableKey: "true"}}
			Expect(getBoolValueForKey(cm, tracingEnableKey)).To(BeTrue())
			Expect(getBoolValueForKey(cm, egressKey)).To(BeFalse())
		})

		It("Test getIntValueForKey()", func() {
			cm := &v1.ConfigMap{Data: map[string]string{tracingPortKey: "12345"}}
			Expect(getIntValueForKey(cm, tracingPortKey)).To(Equal(12345))

			cm0 := &v1.ConfigMap{Data: map[string]string{}}
			Expect(getIntValueForKey(cm0, egressKey)).To(Equal(0))
		})

		It("Test getStringValueForKey()", func() {
			cm := &v1.ConfigMap{Data: map[string]string{tracingEndpointKey: "foo"}}
			Expect(getStringValueForKey(cm, tracingEndpointKey)).To(Equal("foo"))

			cm0 := &v1.ConfigMap{Data: map[string]string{}}
			Expect(getStringValueForKey(cm0, tracingEndpointKey)).To(Equal(""))
		})
	})
})
