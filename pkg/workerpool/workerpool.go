// Package workerpool implements the thread-pool paradigm
// in Go. The benefits of it in Go however, can be quite different
// from any other language able to schedule itself on system threads.
//
// By using a workpool model, the main focus and intention is to limit the
// number of go routines that can do busy-work and get scheduled concurrenly
// at any point in time.
//
// Too many go routines being scheduled at the same time will cause other
// go routines (maybe more critical ones) to be scheduled less often, thus
// incurring in resource starvation on those and potentially triggering other
// issues.
//
// By being able to queue up work, we should be able to run a more deterministic
// runtime (despite Go's nature, this we will not be able to help), less dependant
// on the scheduler and more accurate in terms of time, as now the number of routines
// doing busy work can remain constant as opposed have O(N) routines attempting to run
// at the same time.
package workerpool

import (
	"runtime"
	"sync"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("workerpool")
)

// WorkerPool object representation
type WorkerPool struct {
	wg       sync.WaitGroup // Sync group, to stop workers if needed
	nWorkers uint64         // Number of workers. Uint64 for easier mod hash later
	jobs     chan func()
	stop     chan struct{} // Stop channel
}

// NewWorkerPool creates a new work group.
// If nWorkers is 0, will poll goMaxProcs to get the number of routines to spawn.
// Reminder: routines are never pinned to system threads, it's up to the go scheduler to decide
// when and where these will be scheduled.
func NewWorkerPool(nWorkers int) *WorkerPool {
	if nWorkers == 0 {
		// read GOMAXPROCS, -1 to avoid changing it
		nWorkers = runtime.GOMAXPROCS(-1)
	}

	log.Info().Msgf("New worker pool setting up %d workers", nWorkers)

	workPool := &WorkerPool{
		nWorkers: uint64(nWorkers),
		jobs:     make(chan func(), nWorkers),
		stop:     make(chan struct{}),
	}
	for i := 0; i < nWorkers; i++ {
		i := i
		workPool.wg.Add(1)
		go workPool.work(i)
	}

	return workPool
}

// AddJob posts the job on a worker queue
// Uses Hash underneath to choose worker to post the job to
func (wp *WorkerPool) AddJob(f func()) {
	wp.jobs <- f
}

// GetWorkerNumber get number of queues/workers
func (wp *WorkerPool) GetWorkerNumber() int {
	return int(wp.nWorkers)
}

// Stop stops the workerpool
func (wp *WorkerPool) Stop() {
	close(wp.stop)
	wp.wg.Wait()
}

func (wp *WorkerPool) work(id int) {
	defer wp.wg.Done()

	log.Info().Msgf("Worker %d running", id)

	for {
		select {
		case f := <-wp.jobs:
			f()
		case <-wp.stop:
			log.Debug().Msgf("work[%d]: Stopped", id)
			return
		}
	}
}
