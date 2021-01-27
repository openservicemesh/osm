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

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

const (
	osmNamespace     = "-test-osm-namespace-"
	osmConfigMapName = "-test-osm-config-map-"
)

var _ = Describe("Test OSM ConfigMap parsing", func() {
	defer GinkgoRecover()

	kubeClient := testclient.NewSimpleClientset()

	stop := make(<-chan struct{})
	cfg := newConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
	Expect(cfg).ToNot(BeNil())

	confChannel := events.GetPubSubInstance().Subscribe(
		announcements.ConfigMapAdded,
		announcements.ConfigMapDeleted,
		announcements.ConfigMapUpdated)
	defer events.GetPubSubInstance().Unsub(confChannel)

	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: osmNamespace,
			Name:      osmConfigMapName,
		},
	}
	if _, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("[TEST] Error creating ConfigMap %s/%s/: %s", configMap.Namespace, configMap.Name, err.Error())
	}
	<-confChannel

	Context("Ensure we are able to get reasonable defaults from ConfigMap", func() {

		It("Tag matches const key for all fields of OSM ConfigMap struct", func() {
			fieldNameTag := map[string]string{
				"PermissiveTrafficPolicyMode":  PermissiveTrafficPolicyModeKey,
				"Egress":                       egressKey,
				"EnableDebugServer":            enableDebugServer,
				"PrometheusScraping":           prometheusScrapingKey,
				"TracingEnable":                tracingEnableKey,
				"TracingAddress":               tracingAddressKey,
				"TracingPort":                  tracingPortKey,
				"TracingEndpoint":              tracingEndpointKey,
				"UseHTTPSIngress":              useHTTPSIngressKey,
				"EnvoyLogLevel":                envoyLogLevel,
				"ServiceCertValidityDuration":  serviceCertValidityDurationKey,
				"OutboundIPRangeExclusionList": outboundIPRangeExclusionListKey,
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
					Equal(PermissiveTrafficPolicyModeKey),
					fmt.Sprintf("Field %s expected to have tag %s; fuond %s instead", fieldName, expectedTag, actualtag))
			}
		})

		It("Test GetBoolValueForKey()", func() {
			cm := &v1.ConfigMap{Data: map[string]string{tracingEnableKey: "true"}}
			val, err := GetBoolValueForKey(cm, tracingEnableKey)
			Expect(val).To(BeTrue())
			Expect(err).To(BeNil())

			val, err = GetBoolValueForKey(cm, egressKey)
			Expect(val).To(BeFalse())
			Expect(err).To(HaveOccurred())
		})

		It("Test GetIntValueForKey()", func() {
			cm := &v1.ConfigMap{Data: map[string]string{tracingPortKey: "12345"}}
			val, err := GetIntValueForKey(cm, tracingPortKey)
			Expect(val).To(Equal(12345))
			Expect(err).To(BeNil())

			cm0 := &v1.ConfigMap{Data: map[string]string{}}
			val, err = GetIntValueForKey(cm0, egressKey)
			Expect(val).To(Equal(0))
			Expect(err).To(HaveOccurred())
		})

		It("Test GetStringValueForKey()", func() {
			cm := &v1.ConfigMap{Data: map[string]string{tracingEndpointKey: "foo"}}
			val, err := GetStringValueForKey(cm, tracingEndpointKey)
			Expect(val).To(Equal("foo"))
			Expect(err).To(BeNil())

			cm0 := &v1.ConfigMap{Data: map[string]string{}}
			strval, err := GetStringValueForKey(cm0, tracingEndpointKey)
			Expect(strval).To(Equal(""))
			Expect(err).To(HaveOccurred())
		})
	})
})
