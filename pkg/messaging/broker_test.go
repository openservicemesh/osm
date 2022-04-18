package messaging

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
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

	endpointsChan := c.GetKubeEventPubSub().Sub(
		announcements.EndpointAdded.String(),
		announcements.EndpointUpdated.String(),
		announcements.EndpointDeleted.String(),
	)
	defer c.Unsub(c.kubeEventPubSub, endpointsChan)

	meshCfgChan := c.GetKubeEventPubSub().Sub(announcements.MeshConfigUpdated.String())
	defer c.Unsub(c.kubeEventPubSub, meshCfgChan)

	certRotateChan := c.GetCertPubSub().Sub(announcements.CertificateRotated.String())
	defer c.Unsub(c.certPubSub, certRotateChan)

	numEventTriggers := 50
	// Endpoints add/update/delete will result in proxy update events
	numProxyUpdatesPerEventTrigger := 3
	// MeshConfig update events not related to proxy changes and pod events do not trigger proxy update events
	numNonProxyUpdatesPerEventTrigger := 4
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

			epAdd := events.PubSubMessage{
				Kind:   announcements.EndpointAdded,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().Add(epAdd)

			epDel := events.PubSubMessage{
				Kind:   announcements.EndpointDeleted,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().Add(epDel)

			epUpdate := events.PubSubMessage{
				Kind:   announcements.EndpointUpdated,
				OldObj: i,
				NewObj: i,
			}
			c.GetQueue().Add(epUpdate)

			meshCfgUpdate := events.PubSubMessage{
				Kind:   announcements.MeshConfigUpdated,
				OldObj: &configv1alpha3.MeshConfig{},
				NewObj: &configv1alpha3.MeshConfig{},
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

	doneVerifyingEndpointEvents := make(chan struct{})
	go func() {
		// Verify expected number of service events
		numExpectedServiceEvents := numEventTriggers * 3 // 3 == 1 add, 1 delete, 1 update per trigger
		for i := 0; i < numExpectedServiceEvents; i++ {
			<-endpointsChan
		}
		close(doneVerifyingEndpointEvents)
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
	<-doneVerifyingEndpointEvents
	<-doneVerifyingMeshCfgEvents
	<-doneVerifyingCertEvents
	<-doneVerifyingProxyEvents

	a.EqualValues(c.GetTotalQEventCount(), numEventTriggers*(numProxyUpdatesPerEventTrigger+numNonProxyUpdatesPerEventTrigger))
	a.EqualValues(c.GetTotalQProxyEventCount(), numEventTriggers*numProxyUpdatesPerEventTrigger)
}

func TestGetProxyUpdateEvent(t *testing.T) {
	testCases := []struct {
		name          string
		msg           events.PubSubMessage
		expectEvent   bool
		expectedTopic string
	}{
		{
			name: "egress event",
			msg: events.PubSubMessage{
				Kind: announcements.EgressAdded,
			},
			expectEvent:   true,
			expectedTopic: announcements.ProxyUpdate.String(),
		},
		{
			name: "MeshConfig updated to enable permissive mode",
			msg: events.PubSubMessage{
				Kind: announcements.MeshConfigUpdated,
				OldObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						Traffic: configv1alpha3.TrafficSpec{
							EnablePermissiveTrafficPolicyMode: false,
						},
					},
				},
				NewObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						Traffic: configv1alpha3.TrafficSpec{
							EnablePermissiveTrafficPolicyMode: true,
						},
					},
				},
			},
			expectEvent:   true,
			expectedTopic: announcements.ProxyUpdate.String(),
		},
		{
			name: "MeshConfigUpdate event with unexpected object type",
			msg: events.PubSubMessage{
				Kind:   announcements.MeshConfigUpdated,
				OldObj: "unexpected-type",
			},
			expectEvent: false,
		},
		{
			name: "MeshConfig updated with field that does not result in proxy update",
			msg: events.PubSubMessage{
				Kind: announcements.MeshConfigUpdated,
				OldObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						Observability: configv1alpha3.ObservabilitySpec{
							OSMLogLevel: "trace",
						},
					},
				},
				NewObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						Observability: configv1alpha3.ObservabilitySpec{
							OSMLogLevel: "info",
						},
					},
				},
			},
			expectEvent: false,
		},
		{
			name: "MeshConfig update with feature flags results in proxy update",
			msg: events.PubSubMessage{
				Kind: announcements.MeshConfigUpdated,
				OldObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						FeatureFlags: configv1alpha3.FeatureFlags{
							EnableEgressPolicy: true,
						},
					},
				},
				NewObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						FeatureFlags: configv1alpha3.FeatureFlags{
							EnableEgressPolicy: false,
						},
					},
				},
			},
			expectEvent:   true,
			expectedTopic: announcements.ProxyUpdate.String(),
		},
		{
			name: "Namespace event",
			msg: events.PubSubMessage{
				Kind: announcements.NamespaceAdded,
			},
			expectEvent: false,
		},
		{
			name: "Pod add event",
			msg: events.PubSubMessage{
				Kind: announcements.PodAdded,
			},
			expectEvent: false,
		},
		{
			name: "Pod update event not resulting in proxy update",
			msg: events.PubSubMessage{
				Kind: announcements.PodUpdated,
			},
			expectEvent: false,
		},
		{
			// Metrics annotation updates should update the relevant proxy
			name: "Pod update event resulting in proxy update",
			msg: events.PubSubMessage{
				OldObj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{constants.PrometheusScrapeAnnotation: "false"},
						Labels:      map[string]string{constants.EnvoyUniqueIDLabelName: "foo"},
					},
				},
				NewObj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{constants.PrometheusScrapeAnnotation: "true"},
						Labels:      map[string]string{constants.EnvoyUniqueIDLabelName: "foo"},
					},
				},
				Kind: announcements.PodUpdated,
			},
			expectEvent:   true,
			expectedTopic: "proxy:foo",
		},
		{
			name: "Pod delete event",
			msg: events.PubSubMessage{
				Kind: announcements.PodDeleted,
			},
			expectEvent: false,
		},
		{
			name: "Service add event",
			msg: events.PubSubMessage{
				Kind: announcements.ServiceAdded,
			},
			expectEvent: false,
		},
		{
			name: "Service update event",
			msg: events.PubSubMessage{
				Kind: announcements.ServiceUpdated,
			},
			expectEvent: false,
		},
		{
			name: "Service delete event",
			msg: events.PubSubMessage{
				Kind: announcements.ServiceDeleted,
			},
			expectEvent: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual := getProxyUpdateEvent(tc.msg)
			a.Equal(tc.expectEvent, actual != nil)
			if tc.expectEvent {
				a.Equal(tc.expectedTopic, actual.topic)
			}
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
	b.proxyUpdateCh <- proxyUpdateEvent{
		msg:   events.PubSubMessage{Kind: announcements.Kind("sliding-window")},
		topic: announcements.ProxyUpdate.String(),
	}

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
			b.proxyUpdateCh <- proxyUpdateEvent{
				msg:   events.PubSubMessage{Kind: announcements.Kind("max-window")},
				topic: announcements.ProxyUpdate.String(),
			}
			time.Sleep(1 * time.Second)
		}
		// Verify channel close
		close(b.proxyUpdateCh)
	}()

	<-proxyUpdateReceived
	a.EqualValues(b.GetTotalDispatchedProxyEventCount(), 2) // 1 carried over from sliding window test
}

func TestGetPubSubTopicForProxyUUID(t *testing.T) {
	a := assert.New(t)

	a.Equal("proxy:foo", GetPubSubTopicForProxyUUID("foo"))
	a.Equal("proxy:baz", GetPubSubTopicForProxyUUID("baz"))
}

func TestUpdateMetric(t *testing.T) {
	metricsstore.DefaultMetricsStore.Start(
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter,
	)
	defer metricsstore.DefaultMetricsStore.Stop(
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter,
	)

	a := assert.New(t)

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace",
			Labels: map[string]string{
				constants.OSMKubeResourceMonitorAnnotation: "osm",
			},
		},
	}

	namespace2 := namespace
	namespace2.Labels = map[string]string{
		"testlabel": "testvalue",
	}

	testCases := []struct {
		name                   string
		event                  events.PubSubMessage
		expectedNamespaceCount string
	}{
		{
			name: "namespace added event",
			event: events.PubSubMessage{
				Kind:   announcements.NamespaceAdded,
				OldObj: nil,
				NewObj: &namespace,
			},
			expectedNamespaceCount: "1",
		},
		{
			name: "namespace updated event",
			event: events.PubSubMessage{
				Kind:   announcements.NamespaceUpdated,
				OldObj: &namespace,
				NewObj: &namespace2,
			},
			expectedNamespaceCount: "1",
		},
		{
			name: "namespace deleted event",
			event: events.PubSubMessage{
				Kind:   announcements.NamespaceDeleted,
				OldObj: &namespace2,
				NewObj: nil,
			},
			expectedNamespaceCount: "0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updateMetric(tc.event)

			handler := metricsstore.DefaultMetricsStore.Handler()

			req, err := http.NewRequest("GET", "/metrics", nil)
			a.Nil(err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			a.Equal(http.StatusOK, rr.Code)

			expectedResp := fmt.Sprintf(`# HELP osm_resource_namespace_count Represents the number of monitored namespaces in the service mesh
# TYPE osm_resource_namespace_count gauge
osm_resource_namespace_count %s
`, tc.expectedNamespaceCount /* monitored namespace count */)
			a.Contains(rr.Body.String(), expectedResp)
		})
	}
}
