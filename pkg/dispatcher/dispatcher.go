package dispatcher

import (
	"reflect"
	"strings"
	"time"

	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// maxBroadcastDeadlineTime is the max time we will delay a global proxy update
	// if multiple events that would trigger it get coalesced over time.
	maxBroadcastDeadlineTime = 15 * time.Second

	// maxGraceDeadlineTime is the time we will wait for an additional global proxy update
	// trigger if we just received one.
	maxGraceDeadlineTime = 3 * time.Second
)

var log = logger.New("dispatcher")

var (
	// Broadcast is a group of AnnouncementTypes
	Broadcast = []AnnouncementType{ScheduleProxyBroadcast} // Other modules requesting a global envoy update

	// Endpoint is a group of AnnouncementTypes
	Endpoint = []AnnouncementType{EndpointAdded, EndpointDeleted, EndpointUpdated} // endpoint

	// Namespace is a group of AnnouncementTypes
	Namespace = []AnnouncementType{NamespaceAdded, NamespaceDeleted, NamespaceUpdated} // namespace

	// Pod is a group of AnnouncementTypes
	Pod = []AnnouncementType{PodAdded, PodDeleted, PodUpdated} // pod

	// RouteGroup is a group of AnnouncementTypes
	RouteGroup = []AnnouncementType{RouteGroupAdded, RouteGroupDeleted, RouteGroupUpdated} // routegroup

	// Service is a group of AnnouncementTypes
	Service = []AnnouncementType{ServiceAdded, ServiceDeleted, ServiceUpdated} // service

	// ServiceAccount is a group of AnnouncementTypes
	ServiceAccount = []AnnouncementType{ServiceAccountAdded, ServiceAccountDeleted, ServiceAccountUpdated} // serviceaccount

	// TrafficSplit is a group of AnnouncementTypes
	TrafficSplit = []AnnouncementType{TrafficSplitAdded, TrafficSplitDeleted, TrafficSplitUpdated} // traffic split

	// TrafficTarget is a group of AnnouncementTypes
	TrafficTarget = []AnnouncementType{TrafficTargetAdded, TrafficTargetDeleted, TrafficTargetUpdated} // traffic target

	// Ingress is a group of AnnouncementTypes
	Ingress = []AnnouncementType{IngressAdded, IngressDeleted, IngressUpdated} // Ingress

	// TCPRoute is a group of AnnouncementTypes
	TCPRoute = []AnnouncementType{TCPRouteAdded, TCPRouteDeleted, TCPRouteUpdated} // TCProute
)

var allGroups = [][]AnnouncementType{
	Broadcast,
	Endpoint,
	Namespace,
	Pod,
	RouteGroup,
	Service,
	ServiceAccount,
	TrafficSplit,
	TrafficTarget,
	Ingress,
	TCPRoute,
}

func flatten(groups ...[]AnnouncementType) []AnnouncementType {
	var all []AnnouncementType
	for _, group := range groups {
		all = append(all, group...)
	}
	return all
}

// isDeltaUpdate assesses and returns if a pubsub message contains an actual delta in config
func isDeltaUpdate(psubMsg PubSubMessage) bool {
	// TODO(draychev): "updated" needs to be a constant tied to the actual AnnouncementType
	return !(strings.HasSuffix(psubMsg.AnnouncementType.String(), "updated") &&
		reflect.DeepEqual(psubMsg.OldObj, psubMsg.NewObj))
}

// Start launches the dispatcher goroutine.
func Start(stop chan struct{}) {
	// This will be finely tuned in near future, we can instrument other modules
	// to take ownership of certain events, and just notify Start through
	// ScheduleBroadcastUpdate announcement type
	subChannel := GetPubSubInstance().Subscribe(flatten(allGroups...)...)

	// State and channels for event-coalescing
	broadcastScheduled := false
	chanMovingDeadline := make(<-chan time.Time)
	chanMaxDeadline := make(<-chan time.Time)

	// tl;dr "When a broadcast request is scheduled, we will wait (3s) in case we receive another broadcast request
	// during this delay that can be coalesced (and restart the (3s) count if we do) up to a maximum of (15s) delay"

	// When there is no broadcast scheduled (broadcastScheduled == false) we start a max deadline (15s)
	// and a moving deadline (3s) timers.
	// The max deadline (15s) is the guaranteed hard max time we will wait till the next
	// envoy global broadcast is actually published to update all envoys.
	// Max deadline is used to limit the amount of times we might delay issuing the update, as new broadcast
	// requests can keep on delaying the moving deadline potentially forever.
	// The moving deadline resets if a new delta/change/request is detected in the next (3s). This is used to coalesce updates
	// and avoid issuing global envoy reconfiguration at large if new updates are meant to be received shortly after.
	// Either deadline will trigger the broadcast, whichever happens first, given previous conditions.
	// This mechanism is reset when the broadcast is published.

	for {
		select {
		case <-stop:
			log.Info().Msg("Dispatcher is quitting! (via stop channel)")
			return

		case message := <-subChannel:

			// New message from pubsub
			psubMessage, castOk := message.(PubSubMessage)
			if !castOk {
				log.Error().Msgf("Error casting PubSubMessage: %v", psubMessage)
				continue
			}

			// Identify if this is an actual delta, or just resync
			delta := isDeltaUpdate(psubMessage)
			log.Debug().Msgf("%s - delta: %v", psubMessage.AnnouncementType.String(), delta)

			// Schedule an envoy broadcast update if we either:
			// - detected a config delta
			// - another module requested a broadcast through ScheduleProxyBroadcast
			if delta || psubMessage.AnnouncementType == ScheduleProxyBroadcast {
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
			log.Debug().Msgf("[Moving deadline trigger] Broadcast envoy update")
			GetPubSubInstance().Publish(PubSubMessage{
				AnnouncementType: ProxyBroadcast,
			})

			// broadcast done, reset timer channels
			broadcastScheduled = false
			chanMovingDeadline = make(<-chan time.Time)
			chanMaxDeadline = make(<-chan time.Time)

		case <-chanMaxDeadline:
			log.Debug().Msgf("[Max deadline trigger] Broadcast envoy update")
			GetPubSubInstance().Publish(PubSubMessage{
				AnnouncementType: ProxyBroadcast,
			})

			// broadcast done, reset timer channels
			broadcastScheduled = false
			chanMovingDeadline = make(<-chan time.Time)
			chanMaxDeadline = make(<-chan time.Time)
		}
	}
}
