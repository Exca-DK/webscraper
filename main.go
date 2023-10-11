package main

import (
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Exca-DK/webscraper/scraper"
	"github.com/Exca-DK/webscraper/scraper/analytics"
)

var (
	threadsFlag = flag.Int("threads", 1, "specifies how many threads the scraper should utilize for scrapping content.")
	urls        = flag.String("urls", "", "Comma separated list of urls to scrape, eg. --urls=https://www.golang-book.com/books/intro/1,https://www.golang-book.com/books/intro/2")
)

func main() {
	flag.Parse()
	urls := strings.Split(*urls, ",")
	threads := *threadsFlag
	if threads < 1 {
		threads = 1
	}
	fmt.Printf("Scrapping with %v threads for %v links\n", threads, urls)
	scrapper := scraper.NewScrapper().WithThreads(threads)
	scrapper.Start()
	defer scrapper.Stop()

	analyzers := make(map[string]*analytics.WordFrequencyAnalyzer)
	for _, url := range urls {
		analyzers[url] = analytics.NewWordFrequencyAnalyzer(1)
	}

	ts := time.Now()
	var wg sync.WaitGroup
	for key, value := range analyzers {
		wg.Add(1)
		url := key
		analyzer := value
		scrapper.Scrape(url, analyzer)
		go func() {
			result, err := analyzer.Result()
			if err != nil {
				fmt.Printf("Scrape failed. Url: %s, Error: %v\n", url, err.Error())
			} else {
				fmt.Printf("Scrape finished. Url: %s, Result: %v\n", url, result)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Printf("Scraping finished. Elapsed: %v\n", time.Since(ts))
}
