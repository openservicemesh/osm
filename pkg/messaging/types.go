// Package messaging implements the messaging infrastructure between different
// components within the control plane.
package messaging

import (
	"github.com/cskr/pubsub"
	"k8s.io/client-go/util/workqueue"

	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("message-broker")
)

// Broker implements the message broker functionality
type Broker struct {
	queue                          workqueue.RateLimitingInterface
	proxyUpdatePubSub              *pubsub.PubSub
	proxyUpdateCh                  chan proxyUpdateEvent
	kubeEventPubSub                *pubsub.PubSub
	certPubSub                     *pubsub.PubSub
	totalQEventCount               uint64
	totalQProxyEventCount          uint64
	totalDispatchedProxyEventCount uint64
}

// proxyUpdateEvent specifies the PubSubMessage and topic for an event that
// results in a proxy config update
type proxyUpdateEvent struct {
	msg   events.PubSubMessage
	topic string
}
