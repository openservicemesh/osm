package messaging

import (
	"sync/atomic"
	"time"

	"github.com/cskr/pubsub"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
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
		queue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		proxyUpdatePubSub: pubsub.New(0),
		proxyUpdateCh:     make(chan interface{}),
		kubeEventPubSub:   pubsub.New(0),
		certPubSub:        pubsub.New(0),
	}

	go b.runWorkqueueProcessor(stopCh)
	go b.runProxyUpdateDispatcher(stopCh)

	return b
}

// GetProxyUpdatePubSub returns the PubSub instance corresponding to proxy update events
func (b *Broker) GetProxyUpdatePubSub() *pubsub.PubSub {
	return b.proxyUpdatePubSub
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
			for b.processNextItem() {
			}
		},
		time.Second,
		stopCh,
	)
}

// runProxyUpdateDispatcher runs the dispatcher responsible for batching
// proxy update events received in close proximity.
// It batches proxy update events with the use of 2 timers:
// 1. Sliding window timer that resets when a proxy update event is received
// 2. Max window timer that caps the max duration a sliding window can be reset to
// When either of the above timers expire, the proxy update event is published
// on the dedicated pub-sub instance.
func (b *Broker) runProxyUpdateDispatcher(stopCh <-chan struct{}) {
	// batchTimer and maxTimer are updated by the dispatcher routine
	// when events are processed and timeouts expire. They are initialized
	// with a large timeout (a decade) so they don't time out till an event
	// is received.
	noTimeout := 87600 * time.Hour // A decade
	slidingTimer := time.NewTimer(noTimeout)
	maxTimer := time.NewTimer(noTimeout)

	// dispatchPending indicates whether a proxy update event is pending
	// from being published on the pub-sub. A proxy update event will
	// be held for 'proxyUpdateSlidingWindow' duration to be able to
	// coalesce multiple proxy update events within that duration, before
	// it is dispatched on the pub-sub. The 'proxyUpdateSlidingWindow' duration
	// is a sliding window, which means each event received within a window
	// slides the window further ahead in time, up to a max of 'proxyUpdateMaxWindow'.
	//
	// This mechanism is necessary to avoid triggering proxy update pub-sub events in
	// a hot loop, which would otherwise result in CPU spikes on the controller.
	// We want to coalesce as many proxy update events within the 'proxyUpdateMaxWindow'
	// duration.
	dispatchPending := false
	batchCount := 0 // number of proxy update events batched per dispatch

	for {
		var msg events.PubSubMessage

		select {
		case event, ok := <-b.proxyUpdateCh:
			if !ok {
				log.Warn().Msgf("Proxy update event chan closed, exiting dispatcher")
				return
			}

			msg, ok = event.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Expected type PubSubMessage, got %T", msg)
				continue
			}

			if !dispatchPending {
				// No proxy update events are pending send on the pub-sub.
				// Reset the dispatch timers. The events will be dispatched
				// when either of the timers expire.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(proxyUpdateSlidingWindow)
				if !maxTimer.Stop() {
					<-maxTimer.C
				}
				maxTimer.Reset(proxyUpdateMaxWindow)
				dispatchPending = true
				batchCount++
				log.Trace().Msgf("Pending dispatch of msg kind %s", msg.Kind)
			} else {
				// A proxy update event is pending dispatch. Update the sliding window.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(proxyUpdateSlidingWindow)
				batchCount++
				log.Trace().Msgf("Reset sliding window for msg kind %s", msg.Kind)
			}

		case <-slidingTimer.C:
			slidingTimer.Reset(noTimeout) // 'slidingTimer' drained in this case statement
			// Stop and drain 'maxTimer' before Reset()
			if !maxTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-maxTimer.C
			}
			maxTimer.Reset(noTimeout)
			b.proxyUpdatePubSub.Pub(msg, announcements.ProxyUpdate.String())
			atomic.AddUint64(&b.totalDispatchedProxyEventCount, 1)
			metricsstore.DefaultMetricsStore.ProxyBroadcastEventCount.Inc()
			log.Trace().Msgf("Sliding window expired, msg kind %s, batch size %d", msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-maxTimer.C:
			maxTimer.Reset(noTimeout) // 'maxTimer' drained in this case statement
			// Stop and drain 'slidingTimer' before Reset()
			if !slidingTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-slidingTimer.C
			}
			slidingTimer.Reset(noTimeout)
			b.proxyUpdatePubSub.Pub(msg, announcements.ProxyUpdate.String())
			atomic.AddUint64(&b.totalDispatchedProxyEventCount, 1)
			metricsstore.DefaultMetricsStore.ProxyBroadcastEventCount.Inc()
			log.Trace().Msgf("Max window expired, msg kind %s, batch size %d", msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

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
	if shouldUpdateProxy(msg) {
		log.Trace().Msgf("Msg kind %s will update proxies", msg.Kind)
		atomic.AddUint64(&b.totalQProxyEventCount, 1)
		b.proxyUpdateCh <- msg
	}

	// Publish event to other interested clients, e.g. log level changes, debug server on/off etc.
	b.kubeEventPubSub.Pub(msg, msg.Kind.String())

	// Update event metric
	updateMetric(msg)
}

// updateMetric updates metrics related to the event
func updateMetric(msg events.PubSubMessage) {
	// Generic event metric by virtue of having no labels
	metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues("", "").Inc()
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

// shouldUpdateProxy returns a boolean indicating whether the given event should result in a Proxy configuration update
func shouldUpdateProxy(msg events.PubSubMessage) bool {
	switch msg.Kind {
	case
		//
		// K8s native resource events
		//
		// Endpoint event
		announcements.EndpointAdded, announcements.EndpointDeleted, announcements.EndpointUpdated,
		// Pod event
		announcements.PodAdded, announcements.PodDeleted, announcements.PodUpdated,
		// Service event
		announcements.ServiceAdded, announcements.ServiceDeleted, announcements.ServiceUpdated,
		// k8s Ingress event
		announcements.IngressAdded, announcements.IngressDeleted, announcements.IngressUpdated,
		//
		// OSM resource events
		//
		// Egress event
		announcements.EgressAdded, announcements.EgressDeleted, announcements.EgressUpdated,
		// IngressBackend event
		announcements.IngressBackendAdded, announcements.IngressBackendDeleted, announcements.IngressBackendUpdated,
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
		return true

	case announcements.MeshConfigUpdated:
		prevMeshConfig, okPrevCast := msg.OldObj.(*v1alpha1.MeshConfig)
		newMeshConfig, okNewCast := msg.NewObj.(*v1alpha1.MeshConfig)
		if !okPrevCast || !okNewCast {
			log.Error().Msgf("Expected MeshConfig type, got previous=%T, new=%T", okPrevCast, okNewCast)
			return false
		}

		prevSpec := prevMeshConfig.Spec
		newSpec := newMeshConfig.Spec
		// A proxy config update must only be triggered when a MeshConfig field that maps to a proxy config
		// changes.
		if prevSpec.Traffic.EnableEgress != newSpec.Traffic.EnableEgress ||
			prevSpec.Traffic.EnablePermissiveTrafficPolicyMode != newSpec.Traffic.EnablePermissiveTrafficPolicyMode ||
			prevSpec.Traffic.UseHTTPSIngress != newSpec.Traffic.UseHTTPSIngress ||
			prevSpec.Observability.Tracing != newSpec.Observability.Tracing ||
			prevSpec.Traffic.InboundExternalAuthorization.Enable != newSpec.Traffic.InboundExternalAuthorization.Enable ||
			// Only trigger an update on InboundExternalAuthorization field changes if the new spec has the 'Enable' flag set to true.
			(newSpec.Traffic.InboundExternalAuthorization.Enable && (prevSpec.Traffic.InboundExternalAuthorization != newSpec.Traffic.InboundExternalAuthorization)) {
			return true
		}
		return false

	default:
		return false
	}
}
