package common

import (
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"
)

type IncorrectErrIndexError struct{}

func (e *IncorrectErrIndexError) Error() string {
	return "incorrect answer from pool or error index"
}

func NewIncorrectErrIndexError() error {
	return &IncorrectErrIndexError{}
}

type Job struct {
	args []any
}

// WorkerPool represents a pool of goroutines that can execute any function with arguments.
type WorkerPool struct {
	fn          any
	errIdx      int
	workerCount int
	workersWg   *sync.WaitGroup
	closed      *atomic.Bool

	jobsCount *atomic.Int32
	jobsMutex *sync.Mutex
	jobs      chan Job
	errors    chan error
	done      chan struct{}
}

// NewWorkerPool creates a new WorkerPool with the specified number of workers.
func NewWorkerPool(fn any, errIdx, workerCount int) *WorkerPool {
	return &WorkerPool{
		fn:          fn,
		errIdx:      errIdx,
		workerCount: workerCount,
		workersWg:   &sync.WaitGroup{},
		closed:      &atomic.Bool{},
		jobsCount:   &atomic.Int32{},
		jobsMutex:   &sync.Mutex{},
		jobs:        make(chan Job),
		errors:      make(chan error, 1),
		done:        make(chan struct{}, 1),
	}
}

// Start initializes the worker pool and starts processing jobs.
func (wp *WorkerPool) Start() {
	for range wp.workerCount {
		wp.workersWg.Add(1)

		go func() {
			defer wp.workersWg.Done()

			for job := range wp.jobs {
				if wp.closed.Load() {
					break
				}

				if err := wp.executeJob(job); err != nil {
					wp.jobError(err)
				}

				wp.Done()
			}
		}()
	}
}

func (wp *WorkerPool) executeJob(job Job) error {
	res := wp.execute(job.args)

	if res == nil || len(res) <= wp.errIdx {
		return NewIncorrectErrIndexError()
	}

	if res[wp.errIdx] == nil {
		return nil
	}

	if err, ok := res[wp.errIdx].(error); ok {
		return err
	}

	return NewIncorrectErrIndexError()
}

// execute uses reflection to call the function with the provided arguments.
func (wp *WorkerPool) execute(args []any) []any {
	fnValue := reflect.ValueOf(wp.fn)
	if len(args) != fnValue.Type().NumIn() {
		slog.Error("number of arguments does not match function signature")

		return nil
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	resultValues := fnValue.Call(in)

	results := make([]any, len(resultValues))
	for i, result := range resultValues {
		results[i] = result.Interface()
	}

	return results
}

func (wp *WorkerPool) jobError(err error) {
	select {
	case wp.errors <- err:
	default:
	}
}

func (wp *WorkerPool) Add(delta int32) {
	if wp.jobsCount.Add(delta) == delta {
		select {
		case <-wp.done:
		default:
		}
	}
}

func (wp *WorkerPool) Done() {
	if wp.jobsCount.Add(-1) == 0 {
		select {
		case wp.done <- struct{}{}:
		default:
		}
	}
}

// Submit adds a new Job to the pool.
func (wp *WorkerPool) Submit(args ...any) {
	wp.Add(1)

	wp.jobsMutex.Lock()
	defer wp.jobsMutex.Unlock()

	if wp.closed.Load() {
		return
	}

	wp.jobs <- Job{args: args}
}

// WaitOrError waits for all workers or first error.
func (wp *WorkerPool) WaitOrError() error {
	select {
	case err := <-wp.errors:
		return err
	case <-wp.done:
		return nil
	}
}

// Stop waits for all workers to finish and closes the Job channel.
func (wp *WorkerPool) Stop() {
	wp.jobsMutex.Lock()
	close(wp.jobs)
	wp.closed.Store(true)
	wp.jobsMutex.Unlock()

	wp.workersWg.Wait()

	close(wp.errors)
	close(wp.done)
}
