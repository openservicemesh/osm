// Package messaging implements the messaging infrastructure between different
// components within the control plane.
package messaging

import (
	"github.com/cskr/pubsub"
	"k8s.io/client-go/util/workqueue"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("message-broker")
)

// Broker implements the message broker functionality
type Broker struct {
	queue             workqueue.RateLimitingInterface
	proxyUpdatePubSub *pubsub.PubSub
	// channel used to send proxy updates. The messages are coalesced when sent in a tight loop. The string value
	// is only used for logging.
	proxyUpdateCh                  chan string
	kubeEventPubSub                *pubsub.PubSub
	totalQEventCount               uint64
	totalQProxyEventCount          uint64
	totalDispatchedProxyEventCount uint64

	stop <-chan struct{}
}

const (
	// ProxyUpdateTopic is the topic used to send proxy updates
	ProxyUpdateTopic = "proxy-update"
)
