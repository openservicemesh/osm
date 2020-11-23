package events

import (
	"github.com/cskr/pubsub"

	"github.com/openservicemesh/osm/pkg/announcements"
)

const (
	// Default number of events a subscriber channel will buffer
	defaultAnnouncementChannelSize = 512
)

var (
	// Globally accessible instance, through singleton pattern GetPubSubInstance
	pubSubInstance *osmPubsub
)

// Object which implements the PubSub interface
type osmPubsub struct {
	pSub *pubsub.PubSub
}

// Subscribe is the Subscribe implementation for PubSub
func (c *osmPubsub) Subscribe(aTypes ...announcements.AnnouncementType) chan interface{} {
	subTypes := []string{}
	for _, v := range aTypes {
		subTypes = append(subTypes, string(v))
	}

	return c.pSub.Sub(subTypes...)
}

// Publish is the Publish implementation for PubSub
func (c *osmPubsub) Publish(message PubSubMessage) {
	c.pSub.Pub(message, message.AnnouncementType.String())
}

// GetPubSubInstance returns a unique, global scope PubSub interface instance
// Note that spawning the instance is not thread-safe. First call should happen on
// a single-routine context to avoid races.
func GetPubSubInstance() PubSub {
	if pubSubInstance == nil {
		pubSubInstance = &osmPubsub{
			pSub: pubsub.New(defaultAnnouncementChannelSize),
		}
	}
	return pubSubInstance
}
