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

type Scrapper struct {
	// ctx + cancel as signal for stopping WorkPool
	ctx    context.Context
	cancel func()
	done   chan struct{} // Channel for stop sig of scrapper

	targetsCh chan []scrapeTarget // Channel for receving nie urls to scrape
	jobCh     chan job            // Channel for executing scrapping

	// Duration after which the scraper will rescape known websites.
	// If not set then the scraper will visit websie only once.
	evictionRate time.Duration
	maxDepth     int // Max depth for url scrape
	threads      int // How many threads for execution

	jobIndex atomic.Uint64     // How many scrapes done
	pool     *workers.WorkPool // Pool managing jobs

	analyzer analytics.Analyzer // analyzer gathers scrape results

	// jobs that are running or yet to launch
	// required in order to not scrape two same urls concurrently
	activeMu sync.Mutex
	active   map[string]struct{}
}

func NewScrapper(analyzer analytics.Analyzer) *Scrapper {
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
		maxDepth:  3,
		pool:      workers.NewWorkPool(ch),
		analyzer:  analyzer,
		active:    make(map[string]struct{}),
	}
}

func (s *Scrapper) Start() {
	s.pool = s.pool.WithThreads(uint32(s.threads)) // one thread per job
	s.pool.Start(s.ctx)
	for i := 0; i < s.threads; i++ {
		// create a worker that pulls new tasks all the time
		worker := workers.NewWorker(workers.NewJob(fmt.Sprintf("scrape-%v", i), s.taskLoop))
		// add worker
		s.pool.AddWorker(worker)
	}
	go s.eventLoop()
}

func (s *Scrapper) Stop() {
	select {
	case <-s.done:
	default:
		close(s.done)
		s.cancel()
	}
}

func (s *Scrapper) WithThreads(num int) *Scrapper {
	s.threads = num
	return s
}

func (s *Scrapper) WithEviction(duration time.Duration) *Scrapper {
	s.evictionRate = duration
	return s
}

func (s *Scrapper) WithDepth(depth uint) *Scrapper {
	if depth == 0 {
		depth++
	}
	s.maxDepth = int(depth)
	return s
}

func (s *Scrapper) Scrape(url string) {
	select {
	case <-s.done:
	case s.targetsCh <- []scrapeTarget{{url: url}}:
	}
}

func (s *Scrapper) requestScrape(target scrapeTarget) {
	s.requestScrapes([]scrapeTarget{target})
}

func (s *Scrapper) requestScrapes(targets []scrapeTarget) {
	select {
	case <-s.done:
	case s.targetsCh <- targets:
	}
}

func (s *Scrapper) eventLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	retryQueue := make(prims.Queue[scrapeTarget], 0)
	cache := prims.NewSimpleEvictableCache[string, int](func(url string, depth int) {
		retryQueue.Push(scrapeTarget{
			url:   url,
			depth: depth,
		})
	})

	var targets []scrapeTarget
	for {
		select {
		case <-s.done:
			// scrapper stopped
			return
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
			// the loop will receive notification automatically when it expires
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
			cache.AddIfNotSeen(target.url, target.depth, deadline)
		}
		// clear
		targets = targets[len(targets):]
	}
}

func (s *Scrapper) canQueueTarget(t scrapeTarget) bool {
	if !s.ignoreDepth() && t.depth >= s.maxDepth {
		// TODO warn log when logging pkg will be added
		return false
	}

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

func (s *Scrapper) ignoreDepth() bool {
	return s.maxDepth == -1
}
