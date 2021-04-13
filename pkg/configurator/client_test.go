package configurator

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"

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
	kubeClient := testclient.NewSimpleClientset()

	stop := make(chan struct{})
	defer close(stop)
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
				"PermissiveTrafficPolicyMode":   PermissiveTrafficPolicyModeKey,
				"Egress":                        egressKey,
				"EnableDebugServer":             enableDebugServer,
				"PrometheusScraping":            prometheusScrapingKey,
				"TracingEnable":                 tracingEnableKey,
				"TracingAddress":                tracingAddressKey,
				"TracingPort":                   tracingPortKey,
				"TracingEndpoint":               tracingEndpointKey,
				"UseHTTPSIngress":               useHTTPSIngressKey,
				"MaxDataPlaneConnections":       maxDataPlaneConnectionsKey,
				"EnvoyLogLevel":                 envoyLogLevel,
				"ServiceCertValidityDuration":   serviceCertValidityDurationKey,
				"OutboundIPRangeExclusionList":  outboundIPRangeExclusionListKey,
				"EnablePrivilegedInitContainer": enablePrivilegedInitContainer,
				"ConfigResyncInterval":          configResyncInterval,
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

// Tests config map event trigger routine
func TestConfigMapEventTriggers(t *testing.T) {
	assert := tassert.New(t)
	kubeClient := testclient.NewSimpleClientset()

	confChannel := events.GetPubSubInstance().Subscribe(
		announcements.ConfigMapAdded,
		announcements.ConfigMapDeleted,
		announcements.ConfigMapUpdated)
	defer events.GetPubSubInstance().Unsub(confChannel)

	proxyBroadcastChannel := events.GetPubSubInstance().Subscribe(announcements.ScheduleProxyBroadcast)
	defer events.GetPubSubInstance().Unsub(proxyBroadcastChannel)

	stop := make(chan struct{})
	defer close(stop)
	newConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: osmNamespace,
			Name:      osmConfigMapName,
		},
		Data: make(map[string]string),
		// All false
	}

	if _, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Create(context.TODO(), &configMap, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("[TEST] Error creating ConfigMap %s/%s/: %s", configMap.Namespace, configMap.Name, err.Error())
	}

	// ConfigMap Create will generate a ConfigMap notification, and Configurator will issue a ProxyBroadcast for a Create as well
	<-confChannel
	<-proxyBroadcastChannel

	tests := []struct {
		deltaConfigMapContents map[string]string
		expectProxyBroadcast   bool
	}{
		{
			deltaConfigMapContents: map[string]string{
				egressKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaConfigMapContents: map[string]string{
				PermissiveTrafficPolicyModeKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaConfigMapContents: map[string]string{
				useHTTPSIngressKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaConfigMapContents: map[string]string{
				tracingEnableKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaConfigMapContents: map[string]string{
				tracingAddressKey: "jaeger.jagnamespace.cluster.svc.local",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaConfigMapContents: map[string]string{
				tracingEndpointKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaConfigMapContents: map[string]string{
				tracingPortKey: "3521",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaConfigMapContents: map[string]string{
				prometheusScrapingKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaConfigMapContents: map[string]string{
				configResyncInterval: "24h",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaConfigMapContents: map[string]string{
				envoyLogLevel: "warn",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaConfigMapContents: map[string]string{
				enableDebugServer: "true",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaConfigMapContents: map[string]string{
				serviceCertValidityDurationKey: "30h",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaConfigMapContents: map[string]string{
				serviceCertValidityDurationKey: "30h",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaConfigMapContents: map[string]string{
				enablePrivilegedInitContainer: "true",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaConfigMapContents: map[string]string{
				outboundIPRangeExclusionListKey: "true",
			},
			expectProxyBroadcast: false,
		},
	}

	for _, t := range tests {
		// merge configmap
		for mapKey, mapVal := range t.deltaConfigMapContents {
			configMap.Data[mapKey] = mapVal
		}

		_, err := kubeClient.CoreV1().ConfigMaps(osmNamespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
		assert.NoError(err)
		<-confChannel

		proxyEventReceived := false
		select {
		case <-proxyBroadcastChannel:
			proxyEventReceived = true

		case <-time.NewTimer(300 * time.Millisecond).C:
			// one third of a second should be plenty
		}
		assert.Equal(t.expectProxyBroadcast, proxyEventReceived)
	}
}
