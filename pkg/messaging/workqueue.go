package messaging

import (
	"sync/atomic"

	"k8s.io/client-go/util/workqueue"
)

// GetQueue returns the workqueue instance
func (b *Broker) GetQueue() workqueue.Interface {
	return b.queue
}

// GetTotalQEventCount returns the total number of events queued throughout
// the lifetime of the workqueue.
func (b *Broker) GetTotalQEventCount() uint64 {
	return atomic.LoadUint64(&b.totalQEventCount)
}
