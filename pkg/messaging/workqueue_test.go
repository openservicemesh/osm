package messaging

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProcessNextItem(t *testing.T) {
	a := assert.New(t)
	stop := make(chan struct{})
	defer close(stop)

	b := NewBroker(stop)

	// Verify that a non PubSubMessage does not panic
	b.queue.AddRateLimited("string")
	a.Eventually(func() bool {
		return b.GetTotalQEventCount() == 1
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Verify queue shutdown is graceful
	b.queue.ShutDown()
}
