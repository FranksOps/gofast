package engine

import (
	"context"
	"sync"
)

// JobHandler is a function that processes a TransferJob.
type JobHandler func(context.Context, TransferJob) error

// WorkerPool manages a dynamic set of workers processing jobs.
type WorkerPool struct {
	jobChan JobChannel
	handler JobHandler

	ctx    context.Context
	cancel context.CancelFunc

	mu          sync.Mutex
	workers     map[int]chan struct{}
	workerCount int
	nextID      int
	wg          sync.WaitGroup
}

// NewWorkerPool creates a new dynamic worker pool.
func NewWorkerPool(ctx context.Context, jobChan JobChannel, handler JobHandler) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)
	return &WorkerPool{
		jobChan: jobChan,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
		workers: make(map[int]chan struct{}),
	}
}

// SetWorkerCount scales the number of workers up or down gracefully.
func (p *WorkerPool) SetWorkerCount(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for p.workerCount < count {
		p.addWorker()
	}

	for p.workerCount > count {
		p.removeWorker()
	}
}

// WorkerCount returns the current target number of workers.
func (p *WorkerPool) WorkerCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.workerCount
}

func (p *WorkerPool) addWorker() {
	quitChan := make(chan struct{})
	id := p.nextID
	p.nextID++
	p.workers[id] = quitChan
	p.workerCount++
	p.wg.Add(1)

	go func(id int, quit chan struct{}) {
		defer p.wg.Done()
		for {
			// Prioritize quit and context cancellation checking
			select {
			case <-quit:
				return
			case <-p.ctx.Done():
				return
			default:
			}

			select {
			case <-quit:
				// Worker decommissioned gracefully
				return
			case <-p.ctx.Done():
				// Pool stopped, exit
				return
			case job, ok := <-p.jobChan:
				if !ok {
					// Job channel closed, exit
					return
				}
				// Execute the job
				_ = p.handler(p.ctx, job)
			}
		}
	}(id, quitChan)
}

func (p *WorkerPool) removeWorker() {
	// Find arbitrary worker to decommission
	for id, quit := range p.workers {
		close(quit) // Signal the worker to exit gracefully when it finishes current job
		delete(p.workers, id)
		p.workerCount--
		return // Remove only one
	}
}

// Stop initiates termination of all workers and waits for them to exit.
// Jobs currently running might be aborted since the context is cancelled.
func (p *WorkerPool) Stop() {
	p.cancel()
	p.wg.Wait()
}
