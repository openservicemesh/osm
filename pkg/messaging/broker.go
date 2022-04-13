package messaging

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/cskr/pubsub"
	"golang.org/x/sync/singleflight"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

const (
	// proxyUpdateSlidingWindow is the sliding window duration used to batch proxy update events
	proxyUpdateSlidingWindow = 2 * time.Second

	// proxyUpdateMaxWindow is the max window duration used to batch proxy update events, and is
	// the max amount of time a proxy update event can be held for batching before being dispatched.
	proxyUpdateMaxWindow = 10 * time.Second
)

// NewBroker returns a new message broker instance and starts the internal goroutine
// to process events added to the workqueue.
func NewBroker(stopCh <-chan struct{}) *Broker {
	b := &Broker{
		queue:             workqueue.New(),
		proxyUpdatePubSub: pubsub.New(0),
		proxyUpdateCh:     make(chan proxyUpdateEvent),
		kubeEventPubSub:   pubsub.New(0),
		certPubSub:        pubsub.New(0),
	}

	go b.runWorkqueueProcessor(stopCh)
	go b.runProxyUpdateDispatcher(stopCh)

	return b
}

// GetProxyUpdatePubSub returns the PubSub instance corresponding to proxy update events
func (b *Broker) SubscribeProxyUpdates(topics ...string) (<-chan interface{}, func()) {
	ch := b.proxyUpdatePubSub.Sub(topics...)
	return ch, func() {
		b.Unsub(b.proxyUpdatePubSub, ch)
	}
}

// GetKubeEventPubSub returns the PubSub instance corresponding to k8s events
func (b *Broker) GetKubeEventPubSub() *pubsub.PubSub {
	return b.kubeEventPubSub
}

// GetCertPubSub returns the PubSub instance corresponding to certificate events
func (b *Broker) GetCertPubSub() *pubsub.PubSub {
	return b.certPubSub
}

// GetTotalQProxyEventCount returns the total number of events read from the workqueue
// pertaining to proxy updates
func (b *Broker) GetTotalQProxyEventCount() uint64 {
	return atomic.LoadUint64(&b.totalQProxyEventCount)
}

// GetTotalDispatchedProxyEventCount returns the total number of events dispatched
// to subscribed proxies
func (b *Broker) GetTotalDispatchedProxyEventCount() uint64 {
	return atomic.LoadUint64(&b.totalDispatchedProxyEventCount)
}

// runWorkqueueProcessor starts a goroutine to process events from the workqueue until
// signalled to stop on the given channel.
func (b *Broker) runWorkqueueProcessor(stopCh <-chan struct{}) {
	// Start the goroutine workqueue to process kubernetes events
	// The continuous processing of items in the workqueue will run
	// until signalled to stop.
	// The 'wait.Until' helper is used here to ensure the processing
	// of items in the workqueue continues until signalled to stop, even
	// if 'processNextItems()' returns false.
	go wait.Until(
		func() {
			for {
				// Wait for an item to appear in the queue
				item, shutdown := b.queue.Get()
				if shutdown {
					// We'll retry to start the queue in 1 second, unless stopCh is closed.
					log.Info().Msg("Queue shutdown")
					break
				}
				atomic.AddUint64(&b.totalQEventCount, 1)

				msg, ok := item.(events.PubSubMessage)
				if !ok {
					log.Error().Msgf("Received msg of type %T on workqueue, expected events.PubSubMessage", msg)
					b.queue.Done(item)
					return
				}

				b.processEvent(msg)
				b.queue.Done(item)
			}

		},
		time.Second,
		stopCh,
	)
}

// runProxyUpdateDispatcher runs the dispatcher responsible for batching
// proxy update events received in close proximity.
// It batches proxy update events with singleflight
func (b *Broker) runProxyUpdateDispatcher(stopCh <-chan struct{}) {
	var group singleflight.Group
	for {
		select {
		case event, ok := <-b.proxyUpdateCh:
			if !ok {
				log.Warn().Msgf("Proxy update event chan closed, exiting dispatcher")
				return
			}
			go group.Do("global-updater", func() (interface{}, error) {
				// TODO: we somehow need a timeout, it case a single proxy gets stuck (it would do so by starving the workerpool).
				atomic.AddUint64(&b.totalDispatchedProxyEventCount, 1)
				metricsstore.DefaultMetricsStore.ProxyBroadcastEventCount.Inc()
				b.proxyUpdatePubSub.Pub(event.msg, event.topic)
				// TODO: need to 1) enqueue a blocking update to a proxy channel. 2) enqueue a pubsub blocking update to a specific channel.
				return nil, nil
			})

		case <-stopCh:
			log.Info().Msg("Proxy update dispatcher received stop signal, exiting")
			return
		}
	}
}

// processEvent processes an event dispatched from the workqueue.
// It does the following:
// 1. If the event must update a proxy, it publishes a proxy update message
// 2. Processes other internal control plane events
// 3. Updates metrics associated with the event
func (b *Broker) processEvent(msg events.PubSubMessage) {
	log.Trace().Msgf("Processing msg kind: %s", msg.Kind)
	// Update proxies if applicable
	if event := getProxyUpdateEvent(msg); event != nil {
		log.Trace().Msgf("Msg kind %s will update proxies", msg.Kind)
		atomic.AddUint64(&b.totalQProxyEventCount, 1)
		if event.topic == announcements.ProxyUpdate.String() {
			// Pass the broadcast event to the dispatcher routine, that coalesces
			// multiple broadcasts received in close proximity.
			b.proxyUpdateCh <- *event
		} else {
			// This is not a broadcast event, so it cannot be coalesced with
			// other events as the event is specific to one or more proxies.
			b.proxyUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedProxyEventCount, 1)
		}
	}

	// Publish event to other interested clients, e.g. log level changes, debug server on/off etc.
	b.kubeEventPubSub.Pub(msg, msg.Kind.String())

	// Update event metric
	updateMetric(msg)
}

// updateMetric updates metrics related to the event
func updateMetric(msg events.PubSubMessage) {
	switch msg.Kind {
	case announcements.NamespaceAdded:
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter.Inc()
	case announcements.NamespaceDeleted:
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter.Dec()
	}
}

// Unsub unsubscribes the given channel from the PubSub instance
func (b *Broker) Unsub(pubSub *pubsub.PubSub, ch chan interface{}) {
	// Unsubscription should be performed from a different goroutine and
	// existing messages on the subscribed channel must be drained as noted
	// in https://github.com/cskr/pubsub/blob/v1.0.2/pubsub.go#L95.
	go pubSub.Unsub(ch)
	for range ch {
		// Drain channel until 'Unsub' results in a close on the subscribed channel
	}
}

// getProxyUpdateEvent returns a proxyUpdateEvent type indicating whether the given PubSubMessage should
// result in a Proxy configuration update on an appropriate topic. Nil is returned if the PubSubMessage
// does not result in a proxy update event.
func getProxyUpdateEvent(msg events.PubSubMessage) *proxyUpdateEvent {
	switch msg.Kind {
	case
		//
		// K8s native resource events
		//
		// Endpoint event
		announcements.EndpointAdded, announcements.EndpointDeleted, announcements.EndpointUpdated,
		// k8s Ingress event
		announcements.IngressAdded, announcements.IngressDeleted, announcements.IngressUpdated,
		//
		// OSM resource events
		//
		// Egress event
		announcements.EgressAdded, announcements.EgressDeleted, announcements.EgressUpdated,
		// IngressBackend event
		announcements.IngressBackendAdded, announcements.IngressBackendDeleted, announcements.IngressBackendUpdated,
		// Retry event
		announcements.RetryPolicyAdded, announcements.RetryPolicyDeleted, announcements.RetryPolicyUpdated,
		// UpstreamTrafficSetting event
		announcements.UpstreamTrafficSettingAdded, announcements.UpstreamTrafficSettingDeleted, announcements.UpstreamTrafficSettingUpdated,
		// MulticlusterService event
		announcements.MultiClusterServiceAdded, announcements.MultiClusterServiceDeleted, announcements.MultiClusterServiceUpdated,
		//
		// SMI resource events
		//
		// SMI HTTPRouteGroup event
		announcements.RouteGroupAdded, announcements.RouteGroupDeleted, announcements.RouteGroupUpdated,
		// SMI TCPRoute event
		announcements.TCPRouteAdded, announcements.TCPRouteDeleted, announcements.TCPRouteUpdated,
		// SMI TrafficSplit event
		announcements.TrafficSplitAdded, announcements.TrafficSplitDeleted, announcements.TrafficSplitUpdated,
		// SMI TrafficTarget event
		announcements.TrafficTargetAdded, announcements.TrafficTargetDeleted, announcements.TrafficTargetUpdated,
		//
		// Proxy events
		//
		announcements.ProxyUpdate:
		return &proxyUpdateEvent{
			msg:   msg,
			topic: announcements.ProxyUpdate.String(),
		}

	case announcements.MeshConfigUpdated:
		prevMeshConfig, okPrevCast := msg.OldObj.(*configv1alpha2.MeshConfig)
		newMeshConfig, okNewCast := msg.NewObj.(*configv1alpha2.MeshConfig)
		if !okPrevCast || !okNewCast {
			log.Error().Msgf("Expected MeshConfig type, got previous=%T, new=%T", okPrevCast, okNewCast)
			return nil
		}

		prevSpec := prevMeshConfig.Spec
		newSpec := newMeshConfig.Spec
		// A proxy config update must only be triggered when a MeshConfig field that maps to a proxy config
		// changes.
		if prevSpec.Traffic.EnableEgress != newSpec.Traffic.EnableEgress ||
			prevSpec.Traffic.EnablePermissiveTrafficPolicyMode != newSpec.Traffic.EnablePermissiveTrafficPolicyMode ||
			prevSpec.Observability.Tracing != newSpec.Observability.Tracing ||
			prevSpec.Traffic.InboundExternalAuthorization.Enable != newSpec.Traffic.InboundExternalAuthorization.Enable ||
			// Only trigger an update on InboundExternalAuthorization field changes if the new spec has the 'Enable' flag set to true.
			(newSpec.Traffic.InboundExternalAuthorization.Enable && (prevSpec.Traffic.InboundExternalAuthorization != newSpec.Traffic.InboundExternalAuthorization)) ||
			prevSpec.FeatureFlags != newSpec.FeatureFlags {
			return &proxyUpdateEvent{
				msg:   msg,
				topic: announcements.ProxyUpdate.String(),
			}
		}
		return nil

	case announcements.PodUpdated:
		// Only trigger a proxy update for proxies associated with this pod based on the proxy UUID
		prevPod, okPrevCast := msg.OldObj.(*corev1.Pod)
		newPod, okNewCast := msg.NewObj.(*corev1.Pod)
		if !okPrevCast || !okNewCast {
			log.Error().Msgf("Expected *Pod type, got previous=%T, new=%T", okPrevCast, okNewCast)
			return nil
		}
		prevMetricAnnotation := prevPod.Annotations[constants.PrometheusScrapeAnnotation]
		newMetricAnnotation := newPod.Annotations[constants.PrometheusScrapeAnnotation]
		if prevMetricAnnotation != newMetricAnnotation {
			proxyUUID := newPod.Labels[constants.EnvoyUniqueIDLabelName]
			return &proxyUpdateEvent{
				msg:   msg,
				topic: GetPubSubTopicForProxyUUID(proxyUUID),
			}
		}
		return nil

	default:
		return nil
	}
}

// GetPubSubTopicForProxyUUID returns the topic on which PubSubMessages specific to a proxy UUID are published
func GetPubSubTopicForProxyUUID(uuid string) string {
	return fmt.Sprintf("proxy:%s", uuid)
}
