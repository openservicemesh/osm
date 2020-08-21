package catalog

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/openservicemesh/osm/pkg/configurator"
)

type Broadcaster struct {
	tm            *time.Ticker
	lock          *sync.Mutex
	localStop     chan struct{}
	currentTicker uint64

	cfg            configurator.Configurator
	announcementCh chan interface{}
	stop           <-chan struct{}
}

func NewBroadcaster(cfg configurator.Configurator, stop <-chan struct{}) *Broadcaster {
	b := &Broadcaster{
		lock:          &sync.Mutex{},
		localStop:     make(chan struct{}),
		currentTicker: 0,

		cfg:            cfg,
		announcementCh: make(chan interface{}),
		stop:           stop,
	}

	go b.configWatcher()

	return b
}

// GetAnnouncementsChannel returns the channel on which the broadcaster makes announcements
func (b *Broadcaster) GetAnnouncementsChannel() <-chan interface{} {
	return b.announcementCh
}

func (b *Broadcaster) Reset(newInterval time.Duration) {
	b.lock.Lock()

	if b.tm != nil {
		b.tm.Stop()
		b.localStop <- struct{}{}
		b.tm = nil
	}

	if newInterval == 0 {
		b.lock.Unlock()
		return
	}

	newTickerID := atomic.AddUint64(&b.currentTicker, 1)
	newTicker := time.NewTicker(newInterval)
	b.tm = newTicker

	b.lock.Unlock()

	go func() {
		for {
			select {
			case <-b.localStop:
				newTicker.Stop()
				return
			case <-newTicker.C:
				currentTickerID := atomic.LoadUint64(&b.currentTicker)
				if newTickerID != currentTickerID {
					return
				}

				b.announcementCh <- "[broadcast] periodic announcement"
			}
		}
	}()
}

func (b *Broadcaster) configWatcher() {
	var t time.Duration

	for {
		newT := durationInMinutes(b.cfg.BroadcastEvery())
		if newT != t {
			b.Reset(newT)
		}
		t = newT
	}
}

func durationInMinutes(t int) time.Duration {
	return time.Duration(t) * time.Minute
}
