package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Exca-DK/webscraper/log"
	"github.com/Exca-DK/webscraper/scraper"
	"github.com/Exca-DK/webscraper/scraper/analytics"
)

var (
	threadsFlag = flag.Int("threads", 1, "specifies how many threads the scraper should utilize for scrapping content.")
	urlsFlag    = flag.String("urls", "", "Comma separated list of urls to scrape, eg. --urls=https://www.golang-book.com/books/intro/1,https://www.golang-book.com/books/intro/2")
	lvlFlag     = flag.String("verbosity", log.Info.String(), fmt.Sprintf("specifies the logger output lvl. possible options are: %v", log.Lvls()))
)

func main() {
	flag.Parse()
	var logLvl log.LogLvl
	if err := logLvl.FromString(*lvlFlag); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	logger := log.NewLogger(logLvl, os.Stdout)

	urls := strings.Split(*urlsFlag, ",")
	threads := *threadsFlag
	if threads < 1 {
		threads = 1
	}
	logger.Info("Initializing scrapper.", "threads:", threads, "urls:", urls)
	scrapper := scraper.NewScrapper(logger).WithThreads(threads)
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
				logger.Warn("Scrape failed.", "url:", url, "err:", err.Error())
			} else {
				logger.Info("Scrape finished.", "url:", url, "result:", result)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	logger.Info("Scraping finished.", "duration:", time.Since(ts))
}
