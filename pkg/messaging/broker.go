package messaging

import (
	"fmt"
	"sync/atomic"

	"github.com/cskr/pubsub"
	corev1 "k8s.io/api/core/v1"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// NewBroker returns a new message broker instance and starts the internal goroutine
// to process events added to the workqueue.
func NewBroker(stopCh <-chan struct{}) *Broker {
	return &Broker{
		proxyUpdatePubSub: pubsub.New(0),
		kubeEventPubSub:   pubsub.New(0),
		stop:              stopCh,
	}
}

// Done returns whether the broker has completed processing events.
func (b *Broker) Done() <-chan struct{} {
	return b.stop
}

// SubscribeProxyUpdates subscribes to proxy update topics, and returns an unsubscribe function.
func (b *Broker) SubscribeProxyUpdates(topics ...string) (chan interface{}, func()) {
	ch := b.proxyUpdatePubSub.Sub(topics...)
	return ch, func() {
		b.unsub(b.proxyUpdatePubSub, ch)
	}
}

// SubscribeKubeEvents subscribes to kubernetes events, along with an unsubscribe function.
func (b *Broker) SubscribeKubeEvents(topics ...string) (chan interface{}, func()) {
	ch := b.kubeEventPubSub.Sub(topics...)
	return ch, func() {
		b.unsub(b.kubeEventPubSub, ch)
	}
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

// GetTotalQEventCount returns the total number of events queued throughout
// the lifetime of the workqueue.
func (b *Broker) GetTotalQEventCount() uint64 {
	return atomic.LoadUint64(&b.totalQEventCount)
}

// BroadcastProxyUpdate enqueues a broadcast to update all proxies.
func (b *Broker) BroadcastProxyUpdate() {
	b.AddEvent(events.PubSubMessage{Kind: events.ProxyUpdate, Type: events.Added})
}

// AddEvent processes an event, checking its type and distributing to the appropriate queues.
// It does the following:
// 1. If the event must update a proxy, it publishes a proxy update message
// 2. Processes other internal control plane events
// 3. Updates metrics associated with the event
func (b *Broker) AddEvent(msg events.PubSubMessage) {
	log.Trace().Msgf("Processing msg kind: %s", msg.Kind)
	// Update proxies if applicable
	publish, uuid := shouldPublish(msg)

	if publish {
		log.Trace().Msgf("Msg kind %s will update proxies", msg.Kind)

		atomic.AddUint64(&b.totalQEventCount, 1)
		atomic.AddUint64(&b.totalDispatchedProxyEventCount, 1)
		if uuid == "" {
			// Pass the broadcast event to the dispatcher routine, that coalesces
			// multiple broadcasts received in close proximity.
			b.proxyUpdatePubSub.Pub(msg, ProxyUpdateTopic)
			metricsstore.DefaultMetricsStore.ProxyBroadcastEventCount.Inc()
		} else {
			// This is not a broadcast event, so it cannot be coalesced with
			// other events as the event is specific to one or more proxies.
			b.proxyUpdatePubSub.Pub(msg, GetPubSubTopicForProxyUUID(uuid))
			atomic.AddUint64(&b.totalDispatchedProxyEventCount, 1)
		}
	}

	// Publish event to other interested clients, e.g. log level changes, debug server on/off etc.
	b.kubeEventPubSub.Pub(msg, msg.Topic())

	// Update event metric
	updateMetric(msg)
}

// updateMetric updates metrics related to the event
func updateMetric(msg events.PubSubMessage) {
	switch msg.Topic() {
	case events.Namespace.Added():
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter.Inc()
	case events.Namespace.Deleted():
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter.Dec()
	}
}

// unsub unsubscribes the given channel from the PubSub instance
func (b *Broker) unsub(pubSub *pubsub.PubSub, ch chan interface{}) {
	// Unsubscription should be performed from a different goroutine and
	// existing messages on the subscribed channel must be drained as noted
	// in https://github.com/cskr/pubsub/blob/v1.0.2/pubsub.go#L95.
	go pubSub.Unsub(ch)
	for range ch {
		// Drain channel until 'Unsub' results in a close on the subscribed channel
	}
}

// shouldPublish returns a boolean, whether the publish will result in a proxy update, along with the UUID of a pod, if
// the update belongs to a specific pod.
func shouldPublish(msg events.PubSubMessage) (bool, string) {
	switch msg.Kind {
	case
		events.Endpoint, events.Ingress,
		events.Egress, events.IngressBackend, events.RetryPolicy, events.UpstreamTrafficSetting,
		events.RouteGroup, events.TCPRoute, events.TrafficSplit, events.TrafficTarget, events.Telemetry,
		events.ProxyUpdate:
		return true, ""

	case events.MeshConfig:
		if msg.Type != events.Updated {
			return false, ""
		}
		prevMeshConfig, okPrevCast := msg.OldObj.(*configv1alpha2.MeshConfig)
		newMeshConfig, okNewCast := msg.NewObj.(*configv1alpha2.MeshConfig)
		if !okPrevCast || !okNewCast {
			log.Error().Msgf("Expected MeshConfig type, got previous=%T, new=%T", okPrevCast, okNewCast)
			return false, ""
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
			return true, ""
		}
		return false, ""

	case events.Pod:
		if msg.Type != events.Updated {
			return false, ""
		}
		// Only trigger a proxy update for proxies associated with this pod based on the proxy UUID
		prevPod, okPrevCast := msg.OldObj.(*corev1.Pod)
		newPod, okNewCast := msg.NewObj.(*corev1.Pod)
		if !okPrevCast || !okNewCast {
			log.Error().Msgf("Expected *Pod type, got previous=%T, new=%T", okPrevCast, okNewCast)
			return false, ""
		}
		prevMetricAnnotation := prevPod.Annotations[constants.PrometheusScrapeAnnotation]
		newMetricAnnotation := newPod.Annotations[constants.PrometheusScrapeAnnotation]
		if prevMetricAnnotation != newMetricAnnotation {
			proxyUUID := newPod.Labels[constants.EnvoyUniqueIDLabelName]
			return true, proxyUUID
		}
		return false, ""

	default:
		return false, ""
	}
}

// GetPubSubTopicForProxyUUID returns the topic on which PubSubMessages specific to a proxy UUID are published
func GetPubSubTopicForProxyUUID(uuid string) string {
	return fmt.Sprintf("proxy:%s", uuid)
}
