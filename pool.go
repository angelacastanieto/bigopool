package gopool

import "sync"

type (
	Job interface {
		Execute() (Result, error)
	}

	Result struct{}

	Worker struct {
		// A pool of workers channels that are registered with the dispatcher
		WorkerPool chan chan Job
		// A channel for receiving a job that was dispatched
		JobChannel chan Job
		// A channel for receiving a worker termination signal
		// (quits after processing)
		quit chan bool

		// where to report errors
		errCh  *chan error
		doneCh *chan bool
	}

	Dispatcher struct {
		JobQueue   chan Job
		MaxWorkers int
		WaitGroup  *sync.WaitGroup
		// A pool of workers channels that are registered with the dispatcher
		WorkerPool chan chan Job
		// Collect errors
		ErrorCh chan error
		DoneCh  chan bool

		Errors []error
	}
)

func NewDispatcher(maxWorkers int, queueSize int) *Dispatcher {
	pool := make(chan chan Job, maxWorkers)
	jobq := make(chan Job, queueSize)
	errors := make(chan error)
	return &Dispatcher{
		JobQueue:   jobq,
		MaxWorkers: maxWorkers,
		WorkerPool: pool,
		WaitGroup:  &sync.WaitGroup{},
		ErrorCh:    errors,
		DoneCh:     make(chan bool),
	}
}

func (d *Dispatcher) Execute(joblist ...Job) {
	d.WaitGroup.Add(len(joblist))
	for _, job := range joblist {
		d.JobQueue <- job
	}
}

func (d *Dispatcher) Wait() {
	d.WaitGroup.Wait()
}

func (d *Dispatcher) Run() {
	// starting n number of workers
	for i := 0; i < d.MaxWorkers; i++ {
		worker := NewWorker(d.WorkerPool, &d.ErrorCh, &d.DoneCh)
		worker.Start()
	}

	// start the dispatcher routine
	go d.dispatch()
	go func() {
		for {
			select {
			case err := <-d.ErrorCh:
				d.Errors = append(d.Errors, err)
			case <-d.DoneCh:
				d.WaitGroup.Done()
			}
		}
	}()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-d.JobQueue:
			// a job request has been received
			go func(job Job) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				jobChannel := <-d.WorkerPool
				// dispatch the job to the worker job channel
				jobChannel <- job
			}(job)
		}
	}
}

// NewWorker creates a new worker that can be registered to a WorkerPool
// and receive jobs
func NewWorker(workerPool chan chan Job, errCh *chan error, doneCh *chan bool) Worker {
	return Worker{
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool),
		errCh:      errCh,
		doneCh:     doneCh,
	}
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w Worker) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				_, err := job.Execute()
				if err != nil {
					*w.errCh <- err
				}
				*w.doneCh <- true
			case <-w.quit:
				// we have received a signal to stop
				return
			}
		}
	}()
}
