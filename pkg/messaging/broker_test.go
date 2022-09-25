package messaging

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

func TestAllEvents(t *testing.T) {
	a := assert.New(t)
	stopCh := make(chan struct{})
	defer close(stopCh)

	c := NewBroker(stopCh)

	proxyUpdateChan := c.GetProxyUpdatePubSub().Sub(ProxyUpdateTopic)
	defer c.Unsub(c.proxyUpdatePubSub, proxyUpdateChan)

	podChan, unsubPodCH := c.SubscribeKubeEvents(
		events.Pod.Added(),
		events.Pod.Updated(),
		events.Pod.Deleted(),
	)
	defer unsubPodCH()

	endpointsChan, unsubEpsCh := c.SubscribeKubeEvents(
		events.Endpoint.Added(),
		events.Endpoint.Updated(),
		events.Endpoint.Deleted(),
	)
	defer unsubEpsCh()

	meshCfgChan, unsubMshCfg := c.SubscribeKubeEvents(events.MeshConfig.Updated())
	defer unsubMshCfg()

	numEventTriggers := 50
	// Endpoints add/update/delete will result in proxy update events
	numProxyUpdatesPerEventTrigger := 3
	// MeshConfig update events not related to proxy changes and pod events do not trigger proxy update events
	numNonProxyUpdatesPerEventTrigger := 4
	go func() {
		for i := 0; i < numEventTriggers; i++ {
			podAdd := events.PubSubMessage{
				Kind:   events.Pod,
				Type:   events.Added,
				NewObj: i,
			}
			c.AddEvent(podAdd)

			podDel := events.PubSubMessage{
				Kind:   events.Pod,
				Type:   events.Deleted,
				OldObj: i,
			}
			c.AddEvent(podDel)

			podUpdate := events.PubSubMessage{
				Kind:   events.Pod,
				Type:   events.Updated,
				OldObj: i,
				NewObj: i,
			}
			c.AddEvent(podUpdate)

			epAdd := events.PubSubMessage{
				Kind:   events.Endpoint,
				Type:   events.Added,
				NewObj: i,
			}
			c.AddEvent(epAdd)

			epDel := events.PubSubMessage{
				Kind:   events.Endpoint,
				Type:   events.Deleted,
				OldObj: i,
			}
			c.AddEvent(epDel)

			epUpdate := events.PubSubMessage{
				Kind:   events.Endpoint,
				Type:   events.Updated,
				OldObj: i,
				NewObj: i,
			}
			c.AddEvent(epUpdate)

			meshCfgUpdate := events.PubSubMessage{
				Kind:   events.MeshConfig,
				Type:   events.Updated,
				OldObj: &configv1alpha2.MeshConfig{},
				NewObj: &configv1alpha2.MeshConfig{},
			}
			c.AddEvent(meshCfgUpdate)
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
	<-doneVerifyingProxyEvents

	a.EqualValues(c.GetTotalQEventCount(), numEventTriggers*(numProxyUpdatesPerEventTrigger+numNonProxyUpdatesPerEventTrigger))
	a.EqualValues(c.GetTotalQProxyEventCount(), numEventTriggers*numProxyUpdatesPerEventTrigger)
}

func TestGetProxyUpdateEvent(t *testing.T) {
	testCases := []struct {
		name         string
		msg          events.PubSubMessage
		expectEvent  bool
		expectedUUID string
	}{
		{
			name: "egress event",
			msg: events.PubSubMessage{
				Kind: events.Egress,
				Type: events.Added,
			},
			expectEvent: true,
		},
		{
			name: "MeshConfig updated to enable permissive mode",
			msg: events.PubSubMessage{
				Kind: events.MeshConfig,
				Type: events.Updated,
				OldObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Traffic: configv1alpha2.TrafficSpec{
							EnablePermissiveTrafficPolicyMode: false,
						},
					},
				},
				NewObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Traffic: configv1alpha2.TrafficSpec{
							EnablePermissiveTrafficPolicyMode: true,
						},
					},
				},
			},
			expectEvent: true,
		},
		{
			name: "MeshConfigUpdate event with unexpected object type",
			msg: events.PubSubMessage{
				Kind:   events.MeshConfig,
				Type:   events.Updated,
				OldObj: "unexpected-type",
			},
			expectEvent: false,
		},
		{
			name: "MeshConfig updated with field that does not result in proxy update",
			msg: events.PubSubMessage{
				Kind: events.MeshConfig,
				Type: events.Updated,
				OldObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Observability: configv1alpha2.ObservabilitySpec{
							OSMLogLevel: "trace",
						},
					},
				},
				NewObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Observability: configv1alpha2.ObservabilitySpec{
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
				Kind: events.MeshConfig,
				Type: events.Updated,
				OldObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						FeatureFlags: configv1alpha2.FeatureFlags{
							EnableWASMStats: true,
						},
					},
				},
				NewObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						FeatureFlags: configv1alpha2.FeatureFlags{
							EnableWASMStats: false,
						},
					},
				},
			},
			expectEvent: true,
		},
		{
			name: "Namespace event",
			msg: events.PubSubMessage{
				Kind: events.Namespace,
				Type: events.Added,
			},
			expectEvent: false,
		},
		{
			name: "Pod add event",
			msg: events.PubSubMessage{
				Kind: events.Pod,
				Type: events.Added,
			},
			expectEvent: false,
		},
		{
			name: "Pod update event not resulting in proxy update",
			msg: events.PubSubMessage{
				Kind: events.Pod,
				Type: events.Updated,
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
				Kind: events.Pod,
				Type: events.Updated,
			},
			expectEvent:  true,
			expectedUUID: "foo",
		},
		{
			name: "Pod delete event",
			msg: events.PubSubMessage{
				Kind: events.Pod,
				Type: events.Deleted,
			},
			expectEvent: false,
		},
		{
			name: "Service add event",
			msg: events.PubSubMessage{
				Kind: events.Service,
				Type: events.Added,
			},
			expectEvent: false,
		},
		{
			name: "Service update event",
			msg: events.PubSubMessage{
				Kind: events.Service,
				Type: events.Updated,
			},
			expectEvent: false,
		},
		{
			name: "Service delete event",
			msg: events.PubSubMessage{
				Kind: events.Service,
				Type: events.Deleted,
			},
			expectEvent: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			pub, uuid := shouldPublish(tc.msg)
			a.Equal(tc.expectEvent, pub)
			if tc.expectEvent {
				a.Equal(tc.expectedUUID, uuid)
			}
		})
	}
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
				Kind:   events.Namespace,
				Type:   events.Added,
				OldObj: nil,
				NewObj: &namespace,
			},
			expectedNamespaceCount: "1",
		},
		{
			name: "namespace updated event",
			event: events.PubSubMessage{
				Kind:   events.Namespace,
				Type:   events.Updated,
				OldObj: &namespace,
				NewObj: &namespace2,
			},
			expectedNamespaceCount: "1",
		},
		{
			name: "namespace deleted event",
			event: events.PubSubMessage{
				Kind:   events.Namespace,
				Type:   events.Deleted,
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
