package messaging

import (
	"testing"
	"time"

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
			c.GetQueue().Add(podAdd)

			podDel := events.PubSubMessage{
				Kind:   announcements.PodDeleted,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().Add(podDel)

			podUpdate := events.PubSubMessage{
				Kind:   announcements.PodUpdated,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().Add(podUpdate)

			serviceAdd := events.PubSubMessage{
				Kind:   announcements.ServiceAdded,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().Add(serviceAdd)

			serviceDel := events.PubSubMessage{
				Kind:   announcements.ServiceDeleted,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().Add(serviceDel)

			serviceUpdate := events.PubSubMessage{
				Kind:   announcements.ServiceUpdated,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().Add(serviceUpdate)

			meshCfgUpdate := events.PubSubMessage{
				Kind:   announcements.MeshConfigUpdated,
				OldObj: &configv1alpha1.MeshConfig{},
				NewObj: &configv1alpha1.MeshConfig{},
			}
			c.GetQueue().Add(meshCfgUpdate)
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

	doneVerifyingProxyEvents := make(chan struct{})
	go func() {
		// Verify that atleast 1 proxy update pub-sub is received. We only verify one
		// event here because multiple events from the queue could be batched to 1 pub-sub
		// event to reduce proxy broadcast updates.
		<-proxyUpdateChan
		close(doneVerifyingProxyEvents)
	}()

	<-doneVerifyingPodEvents
	<-doneVerifyingServiceEvents
	<-doneVerifyingMeshCfgEvents
	<-doneVerifyingCertEvents
	<-doneVerifyingProxyEvents

	a.EqualValues(c.GetTotalQEventCount(), numEventTriggers*(numProxyUpdatesPerEventTrigger+numNonProxyUpdatesPerEventTrigger))
	a.EqualValues(c.GetTotalQProxyEventCount(), numEventTriggers*numProxyUpdatesPerEventTrigger)
	log.Trace().Msgf("sss batch expected `proxy event total %d", c.GetTotalQProxyEventCount())
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

func TestRunProxyUpdateDispatcher(t *testing.T) {
	a := assert.New(t)
	stopCh := make(chan struct{})
	defer close(stopCh)

	b := NewBroker(stopCh) // this starts runProxyUpdateDispatcher() in a goroutine
	proxyUpdateChan := b.GetProxyUpdatePubSub().Sub(announcements.ProxyUpdate.String())
	defer b.Unsub(b.proxyUpdatePubSub, proxyUpdateChan)

	// Verify sliding window expiry
	b.proxyUpdateCh <- events.PubSubMessage{Kind: announcements.Kind("sliding-window")}

	time.Sleep(proxyUpdateSlidingWindow + 10*time.Millisecond)
	<-proxyUpdateChan
	a.EqualValues(b.GetTotalDispatchedProxyEventCount(), 1)

	// Verify max window expiry
	proxyUpdateReceived := make(chan struct{})
	go func() {
		<-proxyUpdateChan
		close(proxyUpdateReceived)
	}()
	numEvents := 10
	go func() {
		// Sleep for at least 'proxyUpdateMaxWindow' duration (10s), while
		// ensuring sliding window does not expire. 'proxyUpdateSlidingWindow'
		// expires at 2s intervals, so ensure updates are sent within that window
		// via the 1s sleep.
		for i := 0; i < numEvents; i++ {
			log.Trace().Msg("Dispatching event")
			b.proxyUpdateCh <- events.PubSubMessage{Kind: announcements.Kind("max-window")}
			time.Sleep(1 * time.Second)
		}
	}()

	<-proxyUpdateReceived
	a.EqualValues(b.GetTotalDispatchedProxyEventCount(), 2) // 1 carried over from sliding window test

	// Verify incorrect message type is ignored
	b.proxyUpdateCh <- "not-PubSubMessage"
	a.EqualValues(b.GetTotalDispatchedProxyEventCount(), 2) // No new dispatched events

	// Verify channel close
	close(b.proxyUpdateCh)
}
