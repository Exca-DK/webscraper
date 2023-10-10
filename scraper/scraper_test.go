package scraper

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type testingAnalyzer struct {
	wordsMu sync.Mutex
	times   uint
	srcs    []string
	words   []string
}

func (t *testingAnalyzer) Analyze(src string, words []string) {
	t.wordsMu.Lock()
	defer t.wordsMu.Unlock()
	t.times++
	t.srcs = append(t.srcs, src)
	t.words = append(t.words, words...)
}

func newTestServer(data func() []byte, callback func()) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data())
		callback()
	}))
}

// TestScrape is a comprehensive test of the web scraping functionality.
// This test assesses the scraper's ability to perform its core tasks across various web structures and complexities.
func TestScrape(t *testing.T) {
	// TestScrape/recursive tests the recursive web scraping functionality with various thread counts.
	// It focuses on verifying that the web scraper is capable of recursively seeking and scraping new pages,
	// while respecting its own depth limit. This test examines the scraper's behavior under different thread counts
	// to ensure that it correctly follows links to a specified depth. 	Parallelism is not tested here.
	t.Run("recursive", func(t *testing.T) {
		phrase := `
			<!doctype html>
			<html>
			  <head>
				<title>This is the title of the webpage!</title>
			  </head>
			  <body>
				<p>This is an example paragraph. Anything in the <strong>body</strong> tag will appear on the page, just like this <strong>p</strong> tag and its contents.</p>
			  </body>
			  <a href="%s"> link </a>
			</html>
			`
		run := func(t *testing.T, scrapper *Scrapper, serversAmount int, waitAmount int) {
			servers := make([]*httptest.Server, 0)
			var wg sync.WaitGroup
			index := 1
			f := func() []byte {
				if index == len(servers) {
					return nil
				}
				url := fmt.Sprintf(phrase, servers[index].URL)
				index++
				time.Sleep(1 * time.Second)
				return []byte(url)
			}
			wg.Add(waitAmount)
			for i := 1; i < serversAmount; i++ {
				servers = append(servers, newTestServer(f, func() { wg.Done() }))
			}
			scrapper.Scrape(servers[0].URL)
			wg.Wait()
		}
		t.Run("single thread", func(t *testing.T) {
			t.Parallel()
			analyzer := &testingAnalyzer{words: make([]string, 0)}
			scrapper := NewScrapper(analyzer).WithThreads(1).WithDepth(6)
			scrapper.Start()
			run(t, scrapper, 6, 5)
		})
		t.Run("half threads", func(t *testing.T) {
			t.Parallel()
			analyzer := &testingAnalyzer{words: make([]string, 0)}
			scrapper := NewScrapper(analyzer).WithThreads(3).WithDepth(6)
			scrapper.Start()
			run(t, scrapper, 6, 5)
		})
		t.Run("full threads", func(t *testing.T) {
			t.Parallel()
			analyzer := &testingAnalyzer{words: make([]string, 0)}
			scrapper := NewScrapper(analyzer).WithThreads(6).WithDepth(6)
			scrapper.Start()
			run(t, scrapper, 6, 5)
		})

		t.Run("recursive with depth limit", func(t *testing.T) {
			t.Parallel()
			analyzer := &testingAnalyzer{words: make([]string, 0)}
			scrapper := NewScrapper(analyzer).WithThreads(3).WithDepth(3)
			scrapper.Start()
			run(t, scrapper, 6, 3)

			// sleep a little longer so that extra request would be executed
			time.Sleep(1 * time.Second)

			analyzer.wordsMu.Lock()
			defer analyzer.wordsMu.Unlock()
			if int(analyzer.times) != 3 {
				t.Fatal("unexpected requests done", "expected", 3, "got", analyzer.times)
			}
		})
	})

	// TestParallelScraping tests the parallel web scraping functionality with different thread counts.
	// It evaluates the effect of the thread ratio of workers to requests on the throughput.
	// Specifically, it aims to demonstrate that as the thread ratio approaches 1 (each request having its own thread),
	// the faster throughput is achieved in the web scraping process. This test checks only for ratios of range 0-1.
	t.Run("parallel", func(t *testing.T) {
		// HTML content containing multiple links
		phrase := `
		<!doctype html>
		<html>
		  <head>
			<title>This is the title of the webpage!</title>
		  </head>
		  <body>
			<p>This is an example paragraph. Anything in the <strong>body</strong> tag will appear on the page, just like this <strong>p</strong> tag and its contents.</p>
		  </body>
		  <a href="%s"> link </a>
		  <a href="%s"> link </a>
		  <a href="%s"> link </a>
		  <a href="%s"> link </a>
		  <a href="%s"> link </a>
		  <a href="%s"> link </a>
		</html>
		`
		run := func(t *testing.T, scrapper *Scrapper) {
			servers := make([]*httptest.Server, 0)
			var wg sync.WaitGroup
			index := 0
			f := func() []byte {
				if index != 0 {
					return nil
				}
				url := fmt.Sprintf(phrase, servers[1].URL, servers[2].URL, servers[3].URL, servers[4].URL, servers[5].URL, servers[6].URL)
				index++
				time.Sleep(1 * time.Second)
				return []byte(url)
			}

			for i := 0; i < 7; i++ {
				wg.Add(1)
				servers = append(servers, newTestServer(f, func() { wg.Done() }))
			}
			scrapper.Scrape(servers[0].URL)
			wg.Wait()
		}
		previousTime := time.Duration(math.MaxInt64)
		// Test single-threaded scraping (each request shares a thread)
		t.Run("single threads", func(t *testing.T) {
			analyzer := &testingAnalyzer{words: make([]string, 0)}
			scrapper := NewScrapper(analyzer).WithThreads(1).WithDepth(32)

			scrapper.Start()
			ts := time.Now()
			run(t, scrapper)
			previousTime = time.Since(ts)
		})

		// Test half-threaded scraping (each request shares a thread)
		t.Run("half threads", func(t *testing.T) {
			analyzer := &testingAnalyzer{words: make([]string, 0)}
			scrapper := NewScrapper(analyzer).WithThreads(3).WithDepth(32)

			scrapper.Start()
			ts := time.Now()
			run(t, scrapper)
			duration := time.Since(ts)
			estimated := previousTime*3 - 1*time.Second
			if duration > estimated {
				t.Fatal("half threads not fast enough", "got", duration, "want", estimated)
			}
			previousTime = estimated
		})

		// Test full-threaded scraping (each request has its own thread)
		t.Run("full threads", func(t *testing.T) {
			analyzer := &testingAnalyzer{words: make([]string, 0)}
			scrapper := NewScrapper(analyzer).WithThreads(6).WithDepth(32)

			scrapper.Start()
			ts := time.Now()
			run(t, scrapper)
			duration := time.Since(ts)
			estimated := previousTime*2 - 1*time.Second
			if duration > estimated {
				t.Fatal("half threads not fast enough", "got", duration, "want", estimated)
			}
		})
	})
}
