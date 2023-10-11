package workers

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var (
	gos = runtime.GOMAXPROCS(0)
)

// WorkPool represents a concurrent worker pool for managing and executing tasks in parallel.
// It dynamically manages the number of worker threads based on the provided configuration,
// and guarantees the execution of provided tasks exactly once.
type WorkPool struct {
	started atomic.Bool

	// atomic alligned fields
	target          uint32
	activeWorkers   uint32
	finishedWorkers uint32

	// ch fields
	workersFeed chan<- JobStats
	workersIn   chan Worker
	done        chan struct{}

	mu      sync.Mutex // mutex protecting workers
	workers []Worker   // workers

	wg sync.WaitGroup
}

// NewWorkPool creates and initializes a new WorkPool instance.
func NewWorkPool(feedCh chan<- JobStats) *WorkPool {
	return &WorkPool{
		workers:     make([]Worker, 0),
		target:      uint32(gos),
		workersIn:   make(chan Worker),
		done:        make(chan struct{}),
		workersFeed: feedCh,
	}
}

// Start initiates the operation of the WorkPool.
// It ensures that the pool starts only once and that at least one thread is running.
func (p *WorkPool) Start(ctx context.Context) {
	// ensure that pool starts only once
	if !p.started.CompareAndSwap(false, true) {
		return
	}
	// ensure that atleast 1 thread is running
	if p.target == 0 {
		p.target = 1
	}
	go p.monitor(ctx)
}

func (p *WorkPool) spawnWorker(ctx context.Context, worker Worker) {
	atomic.AddUint32(&p.activeWorkers, 1)
	go func() {
		result := worker.Run(ctx)
		atomic.AddUint32(&p.activeWorkers, ^uint32(0))
		atomic.AddUint32(&p.finishedWorkers, 1)
		p.workersFeed <- result
		p.wg.Done()
	}()
}

func (p *WorkPool) popWorker() Worker {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.workers) == 0 {
		return emptyWorker()
	}

	var x Worker
	x, p.workers = p.workers[0], p.workers[1:]

	return x
}

func (p *WorkPool) addWorker(w Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workers = append(p.workers, w)
}

func (p *WorkPool) monitor(ctx context.Context) {
	workerTicker := time.NewTicker(3 * time.Second)
	for {
		active := atomic.LoadUint32(&p.activeWorkers)
		if active < p.target && p.PendingWorkers() > 0 {
			p.spawnWorker(ctx, p.popWorker())
		}

		select {
		case <-ctx.Done():
			close(p.done)
			p.wg.Wait()
			close(p.workersFeed)
			return
		case w := <-p.workersIn:
			p.wg.Add(1)
			p.addWorker(w)
		case <-workerTicker.C:
			for p.PendingWorkers() != 0 {
				active := atomic.LoadUint32(&p.activeWorkers)
				if active >= p.target {
					break
				}
				p.spawnWorker(ctx, p.popWorker())
			}
		}
	}
}

// AddWorker adds a new worker to the WorkPool for task execution.
func (p *WorkPool) AddWorker(w Worker) {
	select {
	case <-p.done:
	case p.workersIn <- w:
	}
}

// RunningWorkers returns the number of currently running workers in the WorkPool.
func (p *WorkPool) RunningWorkers() int {
	return int(atomic.LoadUint32(&p.activeWorkers))
}

// FinishedWorkers returns the number of workers that have completed their tasks in the WorkPool.
func (p *WorkPool) FinishedWorkers() int {
	return int(atomic.LoadUint32(&p.finishedWorkers))
}

// PendingWorkers returns the number of pending workers in the WorkPool.
func (p *WorkPool) PendingWorkers() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.workers)
}

// WithThreads configures the number of worker threads for the WorkPool.
// It allows you to specify the desired number of worker threads to be utilized
// for concurrent task execution. If the provided 'threads' count is zero, it will
// be incremented to one, ensuring that the pool always has at least one thread.
func (p *WorkPool) WithThreads(threads uint32) *WorkPool {
	if threads == 0 {
		threads++
	}
	atomic.StoreUint32(&p.target, threads)
	return p
}
