package configurator

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
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
		caseName             string
		updateMeshConfigSpec func(*v1alpha1.MeshConfigSpec)
		expectProxyBroadcast bool
	}{
		{
			caseName: "EnableEgress",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Traffic.EnableEgress = true
			},
			expectProxyBroadcast: true,
		},
		{
			caseName: "EnablePermissiveTrafficPolicyMode",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Traffic.EnablePermissiveTrafficPolicyMode = true
			},
			expectProxyBroadcast: true,
		},
		{
			caseName: "UseHTTPSIngress",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Traffic.UseHTTPSIngress = true
			},
			expectProxyBroadcast: true,
		},
		{
			caseName: "TracingEnable",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Observability.Tracing.Enable = true
			},
			expectProxyBroadcast: true,
		},
		{
			caseName: "TracingAddress",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Observability.Tracing.Address = "jaeger.jagnamespace.cluster.svc.local"
			},
			expectProxyBroadcast: true,
		},
		{
			caseName: "TracingEndpoint",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Observability.Tracing.Endpoint = "/my/endpoint"
			},
			expectProxyBroadcast: true,
		},
		{
			caseName: "TracingPort",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Observability.Tracing.Port = 3521
			},
			expectProxyBroadcast: true,
		},
		{
			caseName: "SidecarLogLevel",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Sidecar.LogLevel = "warn"
			},
			expectProxyBroadcast: false,
		},
		{
			caseName: "EnableDebugServer",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Observability.EnableDebugServer = true
			},
			expectProxyBroadcast: false,
		},
		{
			caseName: "ServiceCertValidityDuration",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Certificate.ServiceCertValidityDuration = "30h"
			},
			expectProxyBroadcast: false,
		},
		{
			caseName: "EnablePrivilegedInitContainer",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Sidecar.EnablePrivilegedInitContainer = true
			},
			expectProxyBroadcast: false,
		},
		{
			caseName: "OutboundIPRangeExclusionList",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Traffic.OutboundIPRangeExclusionList = []string{"1.2.3.4/24", "10.0.0.1/8"}
			},
			expectProxyBroadcast: false,
		},
		{
			caseName: "OutboundPortExclusionList",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Traffic.OutboundPortExclusionList = []int{7070, 6080}
			},
			expectProxyBroadcast: false,
		},
		{
			caseName: "ConfigResyncInterval",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Sidecar.ConfigResyncInterval = "24h"
			},
			expectProxyBroadcast: false,
		},
		{
			caseName: "InboundExternalAuthorization",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Traffic.InboundExternalAuthorization.Enable = true
			},
			expectProxyBroadcast: true,
		},
		{
			caseName: "osmLogLevel",
			updateMeshConfigSpec: func(spec *v1alpha1.MeshConfigSpec) {
				spec.Observability.OSMLogLevel = "warn"
			},
			expectProxyBroadcast: false,
		},
	}

	for _, tc := range tests {
		// update meshconfig
		tc.updateMeshConfigSpec(&meshConfig.Spec)

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
		assert.Equal(tc.expectProxyBroadcast, proxyEventReceived, tc.caseName)
	}
}
