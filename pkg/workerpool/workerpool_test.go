package workerpool

import (
	"runtime"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestNewWorkerPool(t *testing.T) {
	assert := tassert.New(t)
	wp := NewWorkerPool(0)

	assert.Equal(wp.GetWorkerNumber(), runtime.GOMAXPROCS(-1))
	wp.Stop()

	wp = NewWorkerPool(25)
	assert.Equal(wp.GetWorkerNumber(), 25)
	wp.Stop()
}

// Uses AddJob, which relies on job hash for queue assignment
func TestAddJob(t *testing.T) {
	njobs := 10 // also worker routines
	wp := NewWorkerPool(njobs)

	// Create and add jobs
	chans := make([]chan struct{}, njobs)
	for i := 0; i < njobs; i++ {
		ch := make(chan struct{}, 1)
		chans[i] = ch
		wp.AddJob(func() { ch <- struct{}{} })
	}

	// Verify all jobs ran through the workers
	for i := 0; i < njobs; i++ {
		<-chans[i]
	}

	wp.Stop()
}
