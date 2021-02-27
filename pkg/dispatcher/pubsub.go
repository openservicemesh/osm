package dispatcher

import "github.com/cskr/pubsub"

// Default number of events a subscriber channel will buffer
const defaultAnnouncementChannelSize = 512

// Globally accessible instance, through singleton pattern GetPubSubInstance
var pubSubInstance *osmPubsub

// Object which implements the PubSub interface
type osmPubsub struct {
	pSub *pubsub.PubSub
}

// Subscribe is the Subscribe implementation for PubSub
func (c *osmPubsub) Subscribe(announcementTypes ...AnnouncementType) chan interface{} {
	var subTypes []string
	for _, announcementType := range announcementTypes {
		subTypes = append(subTypes, announcementType.String())
	}

	return c.pSub.Sub(subTypes...)
}

// Publish is the Publish implementation for PubSub
func (c *osmPubsub) Publish(message PubSubMessage) {
	c.pSub.Pub(message, message.AnnouncementType.String())
}

// Unsub is the Unsub implementation for PubSub.
// It is synchronized, upon exit the channel is guaranteed to be both
// unsubscribed to all topics and closed.
// This is a necessary step to guarantee garbage collection
func (c *osmPubsub) Unsubscribe(unsubChan chan interface{}) {
	// implementation has several requirements (including different goroutine context)
	// https://github.com/cskr/pubsub/blob/v1.0.2/pubsub.go#L102

	syncCh := make(chan struct{})
	go func() {
		// This will close the channel on the pubsub backend
		// https://github.com/cskr/pubsub/blob/v1.0.2/pubsub.go#L264
		c.pSub.Unsub(unsubChan)

		for range unsubChan {
			// Drain channel, read til close
		}
		syncCh <- struct{}{}
	}()

	<-syncCh
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
