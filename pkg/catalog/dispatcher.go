package catalog

import (
	"reflect"
	"strings"
	"time"

	a "github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

const (
	maxBroadcastDeadlineTime = 15 * time.Second
	maxGraceDeadlineTime     = 3 * time.Second
)

func (mc *MeshCatalog) dispatcher() {
	// This will be finely tunned in near future, we can instrument other modules
	// to take ownership of certain events, and just notify dispatcher through
	// ScheduleBroadcastUpdate announcement type
	subChannel := events.GetPubSubInstance().Subscribe(
		a.ScheduleBroadcastUpdate,                                // Other modules requesting a global envoy update
		a.ConfigMapAdded, a.ConfigMapDeleted, a.ConfigMapUpdated, // config
		a.EndpointAdded, a.EndpointDeleted, a.EndpointUpdated, // endpoint
		a.NamespaceAdded, a.NamespaceDeleted, a.NamespaceUpdated, // namespace
		a.PodAdded, a.PodDeleted, a.PodUpdated, // pod
		a.RouteGroupAdded, a.RouteGroupDeleted, a.RouteGroupUpdated, // routegroup
		a.ServiceAdded, a.ServiceDeleted, a.ServiceUpdated, // service
		a.TrafficSplitAdded, a.TrafficSplitDeleted, a.TrafficSplitUpdated, // traffic split
		a.TrafficTargetAdded, a.TrafficTargetDeleted, a.TrafficTargetUpdated, // traffic target
		a.BackpressureAdded, a.BackpressureDeleted, a.BackpressureUpdated, // backpressure
		a.IngressAdded, a.IngressDeleted, a.IngressUpdated, // Ingress
		a.TCPRouteAdded, a.TCPRouteDeleted, a.TCPRouteUpdated, // TCProute
	)

	// State and channels for the event-coalescing
	broadcastScheduled := false
	var chanMovingDeadline <-chan time.Time = make(chan time.Time)
	var chanMaxDeadline <-chan time.Time = make(chan time.Time)

	// When there is no broadcast scheduled (broadcastScheduled == false) we start a max deadline (15s)
	// and a moving deadline (3s) timers.
	// The max deadline (15s) is the guaranteed hard max time we will wait til the next
	// envoy broadcast update is actually published.
	// The moving deadline resets if a new delta/change is detected in the next (3s). This is used to coalesce updates
	// and avoid issuing global envoy reconfiguration at large if new updates are meant to be received shortly after.
	// Either deadline will trigger the broadcast, whichever happens first, given previous conditions.
	// This mechanism is reset when the broadcast is published.

	for {
		select {
		case message := <-subChannel:

			// New message from pubsub
			psubMessage, castOk := message.(events.PubSubMessage)
			if !castOk {
				log.Error().Msgf("Error casting PubSubMessage: %v", psubMessage)
				continue
			}

			// Identify if this is an actual delta, or just resync
			delta := !strings.HasSuffix(psubMessage.AnnouncementType.String(), "updated") ||
				!reflect.DeepEqual(psubMessage.OldObj, psubMessage.NewObj)

			log.Debug().Msgf("[Pubsub] %s - delta: %v", psubMessage.AnnouncementType.String(), delta)

			// Schedule an envoy broadcast update if we either:
			// - detected a config delta
			// - another module specifically requested for so
			if delta || psubMessage.AnnouncementType == a.ScheduleBroadcastUpdate {
				if !broadcastScheduled {
					broadcastScheduled = true
					chanMaxDeadline = time.After(maxBroadcastDeadlineTime)
					chanMovingDeadline = time.After(maxGraceDeadlineTime)
				} else {
					// If a broadcast is already scheduled, just reset the moving deadline
					chanMovingDeadline = time.After(maxGraceDeadlineTime)
				}
			} else {
				// Do nothing on non-delta updates
				continue
			}

		// A select-fallthrough doesn't exist, we are copying some code here
		case <-chanMovingDeadline:
			log.Warn().Msgf("[Moving deadline trigger] Broadcast envoy update")
			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: a.EnvoyBroadcast,
			})

			// broadcast done, reset timer channels
			broadcastScheduled = false
			chanMovingDeadline = make(chan time.Time)
			chanMaxDeadline = make(chan time.Time)

		case <-chanMaxDeadline:
			log.Warn().Msgf("[Max deadline trigger] Broadcast envoy update")
			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: a.EnvoyBroadcast,
			})

			// broadcast done, reset timer channels
			broadcastScheduled = false
			chanMovingDeadline = make(chan time.Time)
			chanMaxDeadline = make(chan time.Time)
		}
	}
}
