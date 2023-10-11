package scraper

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

type testingSingleAnalyzer struct {
	page string
	err  error
	wg   sync.WaitGroup
}

func (t *testingSingleAnalyzer) Analyze(page string) {
	t.page = page
	t.wg.Done()
}

func (t *testingSingleAnalyzer) Cancel(err error) {
	t.err = err
	t.wg.Done()
}

type testingCallbackAnalyzer struct {
	page     string
	err      error
	callback func()
}

func (t *testingCallbackAnalyzer) Analyze(page string) {
	t.page = page
	if t.callback != nil {
		t.callback()
	}
}

func (t *testingCallbackAnalyzer) Cancel(err error) {
	t.err = err
	if t.callback != nil {
		t.callback()
	}
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
	// TestParallelScraping tests the parallel web scraping functionality with different thread counts.
	// It evaluates the effect of the thread ratio of workers to requests on the throughput.
	// Specifically, it aims to demonstrate that as the thread ratio approaches 1 (each request having its own thread),
	// the faster throughput is achieved in the web scraping process. This test checks only for ratios of range 0-1.

	// HTML content containing multiple links
	simplePage := `
		<!doctype html>
		<html>
		  <head>
			<title>This is the title of the webpage!</title>
		  </head>
		  <body>
			<p>This is an example paragraph. Anything in the <strong>body</strong> tag will appear on the page, just like this <strong>p</strong> tag and its contents.</p>
		  </body>
		</html>
		`
	t.Run("parallel", func(t *testing.T) {
		// HTML content containing multiple links
		page := `
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
		run := func(t *testing.T, scrapper *Scrapper, analyzer *testingSingleAnalyzer) {
			servers := make([]*httptest.Server, 0)
			f := func() []byte {
				url := fmt.Sprintf(page, servers[0].URL, servers[1].URL, servers[2].URL, servers[3].URL, servers[4].URL, servers[5].URL)
				time.Sleep(1 * time.Second)
				return []byte(url)
			}

			for i := 0; i < 6; i++ {
				servers = append(servers, newTestServer(f, func() {}))
			}
			for _, server := range servers {
				scrapper.Scrape(server.URL, analyzer)
			}
		}
		previousTime := time.Duration(math.MaxInt64)
		// Test single-threaded scraping (each request shares a thread)
		t.Run("single threads", func(t *testing.T) {
			analyzer := &testingSingleAnalyzer{}
			analyzer.wg.Add(6)
			scrapper := NewScrapper().WithThreads(1)

			scrapper.Start()
			ts := time.Now()
			run(t, scrapper, analyzer)
			analyzer.wg.Wait()
			previousTime = time.Since(ts)
		})

		// Test half-threaded scraping (each request shares a thread)
		t.Run("half threads", func(t *testing.T) {
			analyzer := &testingSingleAnalyzer{}
			analyzer.wg.Add(6)
			scrapper := NewScrapper().WithThreads(3)

			scrapper.Start()
			ts := time.Now()
			run(t, scrapper, analyzer)
			analyzer.wg.Wait()
			duration := time.Since(ts)
			estimated := previousTime*3 - 1*time.Second
			if duration > estimated {
				t.Fatal("half threads not fast enough", "got", duration, "want", estimated)
			}
			previousTime = estimated
		})

		// Test full-threaded scraping (each request has its own thread)
		t.Run("full threads", func(t *testing.T) {
			analyzer := &testingSingleAnalyzer{}
			analyzer.wg.Add(6)
			scrapper := NewScrapper().WithThreads(6)

			scrapper.Start()
			ts := time.Now()
			run(t, scrapper, analyzer)
			analyzer.wg.Wait()
			duration := time.Since(ts)
			estimated := previousTime*2 - 1*time.Second
			if duration > estimated {
				t.Fatal("half threads not fast enough", "got", duration, "want", estimated)
			}
		})
	})

	t.Run("Stop", func(t *testing.T) {
		run := func(t *testing.T, scrapper *Scrapper, analyzer *testingCallbackAnalyzer, serversAmount int) {
			servers := make([]*httptest.Server, 0, serversAmount)
			f := func() []byte {
				time.Sleep(1 * time.Second)
				return []byte(simplePage)
			}
			for i := 0; i < serversAmount; i++ {
				servers = append(servers, newTestServer(f, func() {}))
			}
			for _, server := range servers {
				scrapper.Scrape(server.URL, analyzer)
			}
		}

		scrapper := NewScrapper().WithThreads(1)
		called := 0
		ch := make(chan struct{}, 6)
		analyzer := &testingCallbackAnalyzer{callback: func() {
			called++
			ch <- struct{}{}
		}}
		scrapper.Start()
		run(t, scrapper, analyzer, 6)
		<-ch
		scrapper.Stop()
		if called != 1 {
			t.Fatal("unexpected calls amount", called)
		}
	})

	t.Run("Cancellation", func(t *testing.T) {
		scrapper := NewScrapper().WithThreads(1)
		var wg sync.WaitGroup
		wg.Add(1024)
		analyzer := &testingCallbackAnalyzer{callback: func() { wg.Done() }}
		scrapper.Start()
		urls := make([]string, 1024)
		for i := 0; i < len(urls); i++ {
			urls[i] = strconv.Itoa(i)
		}
		scrapper.ScrapeMulti(urls, analyzer)
		scrapper.Stop()
		wg.Wait()
	})
}
