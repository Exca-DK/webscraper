package scraper

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Exca-DK/webscraper/scraper/analytics"
	"github.com/Exca-DK/webscraper/scraper/prims"
	"github.com/Exca-DK/webscraper/workers"
)

// Scrapper is a web scraping tool designed to fetch, analyze, and navigate web content.
// It provides the capability to configure the number of threads
// Scrapes are done in parallel untill the thread limit is hit
type Scrapper struct {
	// ctx + cancel as signal for stopping WorkPool
	ctx    context.Context
	cancel func()
	done   chan struct{} // Channel for stop sig of scrapper

	targetsCh chan []scrapeTarget // Channel for receving new urls to scrape
	jobCh     chan job            // Channel for executing scrapping

	// Duration after which the scraper can rescape known websites.
	// If not set then the scraper will ignore already seen websites for it's whole lifetime.
	evictionRate time.Duration
	threads      int // How many threads for execution

	// How many scrapes done, each new scrape job increments this jobIndex
	jobIndex atomic.Uint64
	pool     *workers.WorkPool // Pool managing jobs

	// jobs that are running or yet to launch
	// required in order to not scrape two same urls concurrently
	activeMu sync.Mutex
	active   map[string]struct{}

	wg sync.WaitGroup // running scrapper threads (eventLoop + jobs)
}

func NewScrapper() *Scrapper {
	ch := make(chan workers.JobStats)
	go func() {
		for range ch {
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	return &Scrapper{
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
		targetsCh: make(chan []scrapeTarget),
		jobCh:     make(chan job),
		pool:      workers.NewWorkPool(ch),
		active:    make(map[string]struct{}),
	}
}

// Start begins the web scraping process by configuring the worker pool, starting it with the specified
// number of threads, and launching worker threads. The method also initiates the event loop to manage
// the scraping tasks.
func (s *Scrapper) Start() {
	s.pool = s.pool.WithThreads(uint32(s.threads)) // one thread per job
	s.pool.Start(s.ctx)
	for i := 0; i < s.threads; i++ {
		// create a worker that pulls new tasks all the time
		worker := workers.NewWorker(workers.NewJob(fmt.Sprintf("scrape-%v", i), s.taskLoop))
		// add worker
		s.pool.AddWorker(worker)
		s.wg.Add(1)
	}
	s.wg.Add(1)
	go s.eventLoop()
}

// Stop gracefully terminates the web scraping process.
func (s *Scrapper) Stop() {
	select {
	case <-s.done:
	default:
		s.cancel()
		close(s.done)
		s.wg.Wait()
	}
}

// WithThreads configures the number of worker threads to use for web scraping tasks.
func (s *Scrapper) WithThreads(num int) *Scrapper {
	s.threads = num
	return s
}

// WithEviction configures the eviction rate for the worker pool, determining how frequently old entries can be rescaped.
// Default value of 0 means that scraper will not try to retry old entry ever.
func (s *Scrapper) WithEviction(duration time.Duration) *Scrapper {
	s.evictionRate = duration
	return s
}

// Scrape add's url to scrapper queue.
func (s *Scrapper) Scrape(url string, analyzer analytics.Analyzer) {
	s.requestScrape([]scrapeTarget{{url: url, analyzer: analyzer}})
}

// Scrape add's urls to scrapper queue. The analyzer will be called once for each of the url.
func (s *Scrapper) ScrapeMulti(urls []string, analyzer analytics.Analyzer) {
	targets := make([]scrapeTarget, len(urls))
	for i, url := range urls {
		targets[i] = scrapeTarget{
			url:      url,
			analyzer: analyzer,
		}
	}
	s.requestScrape(targets)
}

// requestScrape tries to add the targets to the queue.
func (s *Scrapper) requestScrape(targets []scrapeTarget) {
	select {
	case <-s.done:
		// notify analyzer that it won't be executed if scrapper is already stopped.
		for _, target := range targets {
			target.analyzer.Cancel(s.ctx.Err())
		}
	case s.targetsCh <- targets:
	}
}

// eventLoop is a central loop that manages the web scraping process. It handles requests, retries, and cache management
// while coordinating with worker threads. The event loop ensures efficient, concurrent scraping of web content.
func (s *Scrapper) eventLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	retryQueue := make(prims.Queue[scrapeTarget], 0)
	cache := prims.NewSimpleEvictableCache[string, struct{}](func(_ string, _ struct{}) {})

	var targets []scrapeTarget
OUTER:
	for {
		select {
		case <-s.done:
			// scrapper stopped
			break OUTER
		case req := <-s.targetsCh:
			targets = append(targets, req...)
		case <-ticker.C:
			// try to add elems from failed queue
			for target, ok := retryQueue.Pop(); ok; target, ok = retryQueue.Pop() {
				targets = append(targets, target)
			}
		}

		for _, target := range targets {
			// if already in cache, ignore.
			if cache.Seen(target.url) {
				continue
			}
			// not interested at all. ignore
			if !s.canQueueTarget(target) {
				continue
			}
			if !s.tryQueueTarget(target, func() {
				// clear pending from job thread
				s.activeMu.Lock()
				delete(s.active, target.url)
				s.activeMu.Unlock()
			}) {

				// add to retry and remove from active on failure
				retryQueue.Push(target)
				s.activeMu.Lock()
				delete(s.active, target.url)
				s.activeMu.Unlock()
				continue
			}

			// only add to cache when job has been succesfully accepted by worker.
			var deadline time.Time
			if s.evictionRate != 0 {
				deadline = time.Now().Add(s.evictionRate)
			}
			cache.AddIfNotSeen(target.url, struct{}{}, deadline)
		}
		// clear
		targets = targets[len(targets):]
	}

	// cleanup all of the pending analyzers
	for _, target := range targets {
		target.analyzer.Cancel(s.ctx.Err())
	}

	for _, target := range retryQueue {
		target.analyzer.Cancel(s.ctx.Err())
	}
}

// canQueueTarget checks if a given scrape target can be added to the scraping process.
func (s *Scrapper) canQueueTarget(t scrapeTarget) bool {
	// only unique scans at a time
	s.activeMu.Lock()
	if _, ok := s.active[t.url]; ok {
		s.activeMu.Unlock()
		return false
	}
	s.active[t.url] = struct{}{}
	s.activeMu.Unlock()

	return true
}

// tryQueueTarget attempts to add a scrape target to the job channel for processing by worker threads.
func (s *Scrapper) tryQueueTarget(t scrapeTarget, callback func()) bool {
	j := job{
		target:   t,
		callback: callback,
	}
	select {
	// let some of the workers from pool work on that.
	case s.jobCh <- j:
		return true
	// don't block if full
	default:
		return false
	}
}
