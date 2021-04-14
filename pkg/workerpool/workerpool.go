package workerpool

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// Size of the job queue per worker
	maxJobPerWorker = 4096
)

var (
	log = logger.New("workerpool")
)

// worker context for a worker routine
type worker struct {
	id            int
	jobs          chan Job        // Job queue
	stop          chan struct{}   // Stop channel
	wg            *sync.WaitGroup // Pointer to WorkerPool wg
	jobsProcessed uint64          // Jobs processed by this worker
}

// WorkerPool object representation
type WorkerPool struct {
	wg            sync.WaitGroup // Sync group, to stop workers if needed
	workerContext []*worker      // Worker contexts
	nWorkers      uint64         // Number of workers. Uint64 for easier mod hash later
	rRobinCounter uint64         // Used only by the round robin api. Modified atomically on API.
}

// Job is a runnable interface to queue jobs on a WorkerPool
type Job interface {
	// JobName returns the name of the job.
	JobName() string

	// Hash returns a uint64 hash for a job.
	Hash() uint64

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

	var workPool WorkerPool
	for i := 0; i < nWorkers; i++ {
		workPool.workerContext = append(workPool.workerContext,
			&worker{
				id:            i,
				jobs:          make(chan Job, maxJobPerWorker),
				stop:          make(chan struct{}, 1),
				wg:            &workPool.wg,
				jobsProcessed: 0,
			},
		)
		workPool.wg.Add(1)
		workPool.nWorkers++

		go (workPool.workerContext[i]).work()
	}

	return &workPool
}

// AddJob posts the job on a worker queue
// Uses Hash underneath to choose worker to post the job to
func (wp *WorkerPool) AddJob(job Job) <-chan struct{} {
	wp.workerContext[job.Hash()%wp.nWorkers].jobs <- job
	return job.GetDoneCh()
}

// AddJobRoundRobin adds a job in round robin to the queues
// Concurrent calls to AddJobRoundRobin are thread safe and fair
// between each other
func (wp *WorkerPool) AddJobRoundRobin(jobs Job) {
	added := atomic.AddUint64(&wp.rRobinCounter, 1)
	wp.workerContext[added%wp.nWorkers].jobs <- jobs
}

// GetWorkerNumber get number of queues/workers
func (wp *WorkerPool) GetWorkerNumber() int {
	return int(wp.nWorkers)
}

// Stop stops the workerpool
func (wp *WorkerPool) Stop() {
	for _, worker := range wp.workerContext {
		worker.stop <- struct{}{}
	}
	wp.wg.Wait()
}

func (workContext *worker) work() {
	defer workContext.wg.Done()

	log.Info().Msgf("Worker %d running", workContext.id)
	for {
		select {
		case j := <-workContext.jobs:
			t := time.Now()
			log.Debug().Msgf("work[%d]: Starting %v", workContext.id, j.JobName())

			// Run current job
			j.Run()

			log.Debug().Msgf("work[%d][%s] : took %v", workContext.id, j.JobName(), time.Since(t))
			workContext.jobsProcessed++

		case <-workContext.stop:
			log.Debug().Msgf("work[%d]: Stopped", workContext.id)
			return
		}
	}
}
