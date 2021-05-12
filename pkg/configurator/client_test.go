package configurator

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testclient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

const (
	osmNamespace      = "-test-osm-namespace-"
	osmMeshConfigName = "-test-osm-mesh-config-"
)

var _ = Describe("Test OSM MeshConfig parsing", func() {
	meshConfigClientSet := testclient.NewSimpleClientset()

	confChannel := events.GetPubSubInstance().Subscribe(
		announcements.MeshConfigAdded,
		announcements.MeshConfigDeleted,
		announcements.MeshConfigUpdated)
	defer events.GetPubSubInstance().Unsub(confChannel)

	stop := make(chan struct{})
	defer close(stop)

	cfg := newConfigurator(meshConfigClientSet, stop, osmNamespace, osmMeshConfigName)
	Expect(cfg).ToNot(BeNil())

	meshConfig := v1alpha1.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: osmNamespace,
			Name:      osmMeshConfigName,
		},
	}

	if _, err := meshConfigClientSet.ConfigV1alpha1().MeshConfigs(osmNamespace).Create(context.TODO(), &meshConfig, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("[TEST] Error creating MeshConfig %s/%s/: %s", meshConfig.Namespace, meshConfig.Name, err.Error())
	}
	<-confChannel

	Context("Ensure we are able to get reasonable defaults from MeshConfig", func() {

		It("Tag matches const key for all fields of OSM MeshConfig struct", func() {
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
				"EnvoyLogLevel":                 envoyLogLevelKey,
				"EnvoyImage":                    envoyImageKey,
				"InitContainerImage":            initContainerImage,
				"ServiceCertValidityDuration":   serviceCertValidityDurationKey,
				"OutboundIPRangeExclusionList":  outboundIPRangeExclusionListKey,
				"OutboundPortExclusionList":     outboundPortExclusionListKey,
				"EnablePrivilegedInitContainer": enablePrivilegedInitContainerKey,
				"ConfigResyncInterval":          configResyncIntervalKey,
				"MaxDataPlaneConnections":       maxDataPlaneConnectionsKey,
				"ProxyResources":                proxyResourcesKey,
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
					fmt.Sprintf("Field %s expected to have tag %s; found %s instead", fieldName, expectedTag, actualtag))
			}
		})
	})
})

// Tests config map event trigger routine
func TestMeshConfigEventTriggers(t *testing.T) {
	assert := tassert.New(t)
	meshConfigClientSet := testclient.NewSimpleClientset()

	confChannel := events.GetPubSubInstance().Subscribe(
		announcements.MeshConfigAdded,
		announcements.MeshConfigDeleted,
		announcements.MeshConfigUpdated)
	defer events.GetPubSubInstance().Unsub(confChannel)

	proxyBroadcastChannel := events.GetPubSubInstance().Subscribe(announcements.ScheduleProxyBroadcast)
	defer events.GetPubSubInstance().Unsub(proxyBroadcastChannel)

	stop := make(chan struct{})
	defer close(stop)
	_ = newConfigurator(meshConfigClientSet, stop, osmNamespace, meshConfigInformerName)

	meshConfig := v1alpha1.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: osmNamespace,
			Name:      osmMeshConfigName,
		},
	}

	if _, err := meshConfigClientSet.ConfigV1alpha1().MeshConfigs(osmNamespace).Create(context.TODO(), &meshConfig, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("[TEST] Error creating MeshConfig %s/%s/: %s", meshConfig.Namespace, meshConfig.Name, err.Error())
	}

	// MeshConfig Create will generate a MeshConfig notification, and Configurator will issue a ProxyBroadcast for a Create as well
	<-confChannel
	<-proxyBroadcastChannel

	tests := []struct {
		deltaMeshConfigContents map[string]string
		expectProxyBroadcast    bool
	}{
		{
			deltaMeshConfigContents: map[string]string{
				egressKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaMeshConfigContents: map[string]string{
				PermissiveTrafficPolicyModeKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaMeshConfigContents: map[string]string{
				useHTTPSIngressKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaMeshConfigContents: map[string]string{
				tracingEnableKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaMeshConfigContents: map[string]string{
				tracingAddressKey: "jaeger.jagnamespace.cluster.svc.local",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaMeshConfigContents: map[string]string{
				tracingEndpointKey: "true",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaMeshConfigContents: map[string]string{
				tracingPortKey: "3521",
			},
			expectProxyBroadcast: true,
		},
		{
			deltaMeshConfigContents: map[string]string{
				envoyLogLevelKey: "warn",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaMeshConfigContents: map[string]string{
				enableDebugServer: "true",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaMeshConfigContents: map[string]string{
				serviceCertValidityDurationKey: "30h",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaMeshConfigContents: map[string]string{
				serviceCertValidityDurationKey: "30h",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaMeshConfigContents: map[string]string{
				enablePrivilegedInitContainerKey: "true",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaMeshConfigContents: map[string]string{
				outboundIPRangeExclusionListKey: "1.2.3.4/24,10.0.0.1/8",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaMeshConfigContents: map[string]string{
				outboundPortExclusionListKey: "7070, 6080",
			},
			expectProxyBroadcast: false,
		},
		{
			deltaMeshConfigContents: map[string]string{
				configResyncIntervalKey: "24h",
			},
			expectProxyBroadcast: false,
		},
	}

	for _, tc := range tests {
		// merge meshconfig
		for mapKey, mapVal := range tc.deltaMeshConfigContents {
			switch mapKey {
			case egressKey:
				meshConfig.Spec.Traffic.EnableEgress, _ = strconv.ParseBool(mapVal)
			case PermissiveTrafficPolicyModeKey:
				meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode, _ = strconv.ParseBool(mapVal)
			case useHTTPSIngressKey:
				meshConfig.Spec.Traffic.UseHTTPSIngress, _ = strconv.ParseBool(mapVal)
			case tracingEnableKey:
				meshConfig.Spec.Observability.Tracing.Enable, _ = strconv.ParseBool(mapVal)
			case tracingAddressKey:
				meshConfig.Spec.Observability.Tracing.Address = mapVal
			case tracingEndpointKey:
				meshConfig.Spec.Observability.Tracing.Endpoint = mapVal
			case tracingPortKey:
				port, _ := strconv.ParseInt(mapVal, 10, 16)
				meshConfig.Spec.Observability.Tracing.Port = int16(port)
			case envoyLogLevelKey:
				meshConfig.Spec.Sidecar.LogLevel = mapVal
			case enableDebugServer:
				meshConfig.Spec.Observability.EnableDebugServer, _ = strconv.ParseBool(mapVal)
			case serviceCertValidityDurationKey:
				meshConfig.Spec.Certificate.ServiceCertValidityDuration = mapVal
			case enablePrivilegedInitContainerKey:
				meshConfig.Spec.Sidecar.EnablePrivilegedInitContainer, _ = strconv.ParseBool(mapVal)
			case outboundIPRangeExclusionListKey:
				meshConfig.Spec.Traffic.OutboundIPRangeExclusionList = strings.Split(mapVal, ",")
			case outboundPortExclusionListKey:
				portExclusionListStr := strings.Split(mapVal, ",")
				var portExclusionList []int
				for _, portStr := range portExclusionListStr {
					port, _ := strconv.Atoi(portStr)
					portExclusionList = append(portExclusionList, port)
				}
				meshConfig.Spec.Traffic.OutboundPortExclusionList = portExclusionList
			case configResyncIntervalKey:
				meshConfig.Spec.Sidecar.ConfigResyncInterval = mapVal
			}
		}

		_, err := meshConfigClientSet.ConfigV1alpha1().MeshConfigs(osmNamespace).Update(context.TODO(), &meshConfig, metav1.UpdateOptions{})
		assert.NoError(err)
		<-confChannel

		proxyEventReceived := false
		select {
		case <-proxyBroadcastChannel:
			proxyEventReceived = true

		case <-time.NewTimer(300 * time.Millisecond).C:
			// one third of a second should be plenty
		}
		assert.Equal(tc.expectProxyBroadcast, proxyEventReceived)
	}
}
