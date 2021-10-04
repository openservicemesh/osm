package messaging

import (
	"testing"

	"github.com/stretchr/testify/assert"

	configv1alpha1 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s/events"
)

func TestAllEvents(t *testing.T) {
	a := assert.New(t)
	stopCh := make(chan struct{})
	defer close(stopCh)

	c := NewBroker(stopCh)

	proxyUpdateChan := c.GetProxyUpdatePubSub().Sub(announcements.ProxyUpdate.String())
	defer c.Unsub(c.proxyUpdatePubSub, proxyUpdateChan)

	podChan := c.GetKubeEventPubSub().Sub(
		announcements.PodAdded.String(),
		announcements.PodUpdated.String(),
		announcements.PodDeleted.String(),
	)
	defer c.Unsub(c.kubeEventPubSub, podChan)

	serviceChan := c.GetKubeEventPubSub().Sub(
		announcements.ServiceAdded.String(),
		announcements.ServiceUpdated.String(),
		announcements.ServiceDeleted.String(),
	)
	defer c.Unsub(c.kubeEventPubSub, serviceChan)

	meshCfgChan := c.GetKubeEventPubSub().Sub(announcements.MeshConfigUpdated.String())
	defer c.Unsub(c.kubeEventPubSub, meshCfgChan)

	certRotateChan := c.GetCertPubSub().Sub(announcements.CertificateRotated.String())
	defer c.Unsub(c.certPubSub, certRotateChan)

	numEventTriggers := 50
	// 6 messagges pod/service add/update/delete will result in proxy update events
	numProxyUpdatesPerEventTrigger := 6
	// MeshConfig update events not related to proxy change does trigger proxy update events
	numNonProxyUpdatesPerEventTrigger := 1
	go func() {
		for i := 0; i < numEventTriggers; i++ {
			podAdd := events.PubSubMessage{
				Kind:   announcements.PodAdded,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().AddRateLimited(podAdd)

			podDel := events.PubSubMessage{
				Kind:   announcements.PodDeleted,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().AddRateLimited(podDel)

			podUpdate := events.PubSubMessage{
				Kind:   announcements.PodUpdated,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().AddRateLimited(podUpdate)

			serviceAdd := events.PubSubMessage{
				Kind:   announcements.ServiceAdded,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().AddRateLimited(serviceAdd)

			serviceDel := events.PubSubMessage{
				Kind:   announcements.ServiceDeleted,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().AddRateLimited(serviceDel)

			serviceUpdate := events.PubSubMessage{
				Kind:   announcements.ServiceUpdated,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().AddRateLimited(serviceUpdate)

			meshCfgUpdate := events.PubSubMessage{
				Kind:   announcements.MeshConfigUpdated,
				OldObj: &configv1alpha1.MeshConfig{},
				NewObj: &configv1alpha1.MeshConfig{},
			}
			c.GetQueue().AddRateLimited(meshCfgUpdate)
		}
	}()

	go func() {
		for i := 0; i < numEventTriggers; i++ {
			certRotated := events.PubSubMessage{
				Kind:   announcements.CertificateRotated,
				OldObj: i,
				NewObj: i,
			}
			c.certPubSub.Pub(certRotated, announcements.CertificateRotated.String())
		}
	}()

	doneVerifyingProxyEvents := make(chan struct{})
	go func() {
		// Verify expected number of proxy update events are received
		numExpectedBroadcasts := numEventTriggers * numProxyUpdatesPerEventTrigger
		for i := 0; i < numExpectedBroadcasts; i++ {
			<-proxyUpdateChan
		}
		close(doneVerifyingProxyEvents)
	}()

	doneVerifyingPodEvents := make(chan struct{})
	go func() {
		// Verify expected number of pod events
		numExpectedPodevents := numEventTriggers * 3 // 3 == 1 add, 1 delete, 1 update
		for i := 0; i < numExpectedPodevents; i++ {
			<-podChan
		}
		close(doneVerifyingPodEvents)
	}()

	doneVerifyingServiceEvents := make(chan struct{})
	go func() {
		// Verify expected number of service events
		numExpectedServiceEvents := numEventTriggers * 3 // 3 == 1 add, 1 delete, 1 update per trigger
		for i := 0; i < numExpectedServiceEvents; i++ {
			<-serviceChan
		}
		close(doneVerifyingServiceEvents)
	}()

	doneVerifyingMeshCfgEvents := make(chan struct{})
	go func() {
		numExpectedMeshCfgEvents := numEventTriggers * 1 // 1 == 1 update event per trigger
		for i := 0; i < numExpectedMeshCfgEvents; i++ {
			<-meshCfgChan
		}
		close(doneVerifyingMeshCfgEvents)
	}()

	doneVerifyingCertEvents := make(chan struct{})
	go func() {
		numExpectedCertEvents := numEventTriggers * 1 // 1 == 1 cert rotation event per trigger
		for i := 0; i < numExpectedCertEvents; i++ {
			<-certRotateChan
		}
		close(doneVerifyingCertEvents)
	}()

	<-doneVerifyingProxyEvents
	<-doneVerifyingPodEvents
	<-doneVerifyingServiceEvents
	<-doneVerifyingMeshCfgEvents
	<-doneVerifyingCertEvents

	a.EqualValues(c.GetTotalQEventCount(), numEventTriggers*(numProxyUpdatesPerEventTrigger+numNonProxyUpdatesPerEventTrigger))
}

func TestShouldUpdateProxy(t *testing.T) {
	testCases := []struct {
		name     string
		msg      events.PubSubMessage
		expected bool
	}{
		{
			name: "egress event",
			msg: events.PubSubMessage{
				Kind: announcements.EgressAdded,
			},
			expected: true,
		},
		{
			name: "MeshConfig updated to enable permissive mode",
			msg: events.PubSubMessage{
				Kind: announcements.MeshConfigUpdated,
				OldObj: &configv1alpha1.MeshConfig{
					Spec: configv1alpha1.MeshConfigSpec{
						Traffic: configv1alpha1.TrafficSpec{
							EnablePermissiveTrafficPolicyMode: false,
						},
					},
				},
				NewObj: &configv1alpha1.MeshConfig{
					Spec: configv1alpha1.MeshConfigSpec{
						Traffic: configv1alpha1.TrafficSpec{
							EnablePermissiveTrafficPolicyMode: true,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "MeshConfigUpdate event with unexpected object type",
			msg: events.PubSubMessage{
				Kind:   announcements.MeshConfigUpdated,
				OldObj: "unexpected-type",
			},
			expected: false,
		},
		{
			name: "MeshConfig updated with field that does not result in proxy update",
			msg: events.PubSubMessage{
				Kind: announcements.MeshConfigUpdated,
				OldObj: &configv1alpha1.MeshConfig{
					Spec: configv1alpha1.MeshConfigSpec{
						Observability: configv1alpha1.ObservabilitySpec{
							OSMLogLevel: "trace",
						},
					},
				},
				NewObj: &configv1alpha1.MeshConfig{
					Spec: configv1alpha1.MeshConfigSpec{
						Observability: configv1alpha1.ObservabilitySpec{
							OSMLogLevel: "info",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Namespace event",
			msg: events.PubSubMessage{
				Kind: announcements.NamespaceAdded,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual := shouldUpdateProxy(tc.msg)
			a.Equal(tc.expected, actual)
		})
	}
}
