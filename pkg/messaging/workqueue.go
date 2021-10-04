package messaging

import (
	"sync/atomic"

	"k8s.io/client-go/util/workqueue"

	"github.com/openservicemesh/osm/pkg/k8s/events"
)

// GetQueue returns the workqueue instance
func (b *Broker) GetQueue() workqueue.RateLimitingInterface {
	return b.queue
}

// GetTotalQEventCount returns the total number of events queued throughout
// the lifetime of the workqueue.
func (b *Broker) GetTotalQEventCount() uint64 {
	return atomic.LoadUint64(&b.totalQEventCount)
}

// processNextItem processes the next item in the workqueue. It returns a boolean
// indicating if the next item in the queue is ready to be processed.
func (b *Broker) processNextItem() bool {
	// Wait for an item to appear in the queue
	item, shutdown := b.queue.Get()
	if shutdown {
		log.Info().Msg("Queue shutdown")
		return false
	}
	atomic.AddUint64(&b.totalQEventCount, 1)

	// Inform the queue that this 'msg' has been staged for further processing.
	// This is required for safe parallel processing on the queue.
	defer b.queue.Done(item)

	msg, ok := item.(events.PubSubMessage)
	if !ok {
		log.Error().Msgf("Received msg of type %T on workqueue, expected events.PubSubMessage", msg)
		b.queue.Forget(item)
		// Process next item in the queue
		return true
	}

	b.processEvent(msg)
	b.queue.Forget(item)

	return true
}
