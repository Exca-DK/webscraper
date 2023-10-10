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
	pendingMu sync.Mutex
	pending   map[string]struct{}
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
		pending:   make(map[string]struct{}),
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
	cache := prims.NewSimpleEvictableCache[string, int](func(url string, depth int) {
		// don't deadlock
		go s.requestScrape(scrapeTarget{
			url:   "",
			depth: 0,
		})
	})
	for {
		select {
		case <-s.done:
			// scrapper stopped
			return
		case targets := <-s.targetsCh:
			for _, target := range targets {
				// if already in cache, ignore.
				// the loop will receive notification automatically when it expires
				if cache.Seen(target.url) {
					fmt.Println("ignoring seen url", "url", target.url, "depth", target.depth)
					continue
				}
				if s.queueTarget(target, func() {
					// clear pending
					fmt.Println("removing job", "url", target.url, "depth", target.depth)
					s.pendingMu.Lock()
					defer s.pendingMu.Unlock()
					delete(s.pending, target.url)
				}) {
					// only add to cache when job has been succesfully accepted by worker.
					var deadline time.Time
					if s.evictionRate != 0 {
						deadline = time.Now().Add(s.evictionRate)
					}
					cache.AddIfNotSeen(target.url, target.depth, deadline)
					fmt.Println("added url to scrape", "url", target.url, "depth", target.depth)
				}
			}
		}
	}
}

func (s *Scrapper) queueTarget(t scrapeTarget, callback func()) bool {
	if !s.ignoreDepth() && t.depth >= s.maxDepth {
		// TODO warn log when logging pkg will be added
		return false
	}

	// only unique scans at a time
	s.pendingMu.Lock()
	if _, ok := s.pending[t.url]; ok {
		s.pendingMu.Unlock()
		return false
	}
	s.pending[t.url] = struct{}{}
	s.pendingMu.Unlock()

	// let some of the workers from pool work on that.
	s.jobCh <- job{
		target:   t,
		callback: callback,
	}
	return true
}

func (s *Scrapper) ignoreDepth() bool {
	return s.maxDepth == -1
}
