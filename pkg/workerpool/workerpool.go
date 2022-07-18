package workerpool

import (
	"runtime"
	"sync"
	"time"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("workerpool")
)

// WorkerPool object representation
type WorkerPool struct {
	wg       sync.WaitGroup // Sync group, to stop workers if needed
	nWorkers uint64         // Number of workers. Uint64 for easier mod hash later
	jobs     chan Job
	stop     chan struct{} // Stop channel
}

// Job is a runnable interface to queue jobs on a WorkerPool
type Job interface {
	// JobName returns the name of the job.
	JobName() string

	// Run executes the job.
	Run()

	// GetDoneCh returns the channel, which when closed, indicates that the job was finished.
	GetDoneCh() <-chan struct{}
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
		jobs:     make(chan Job, nWorkers),
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
func (wp *WorkerPool) AddJob(job Job) <-chan struct{} {
	wp.jobs <- job
	return job.GetDoneCh()
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
		case j := <-wp.jobs:
			t := time.Now()
			log.Debug().Msgf("work[%d]: Starting %v", id, j.JobName())

			// Run current job
			j.Run()

			log.Debug().Msgf("work[%d][%s] : took %v", id, j.JobName(), time.Since(t))
		case <-wp.stop:
			log.Debug().Msgf("work[%d]: Stopped", id)
			return
		}
	}
}
