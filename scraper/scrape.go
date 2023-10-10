package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Exca-DK/webscraper/scraper/html"
)

// scrapeTarget represents a target for web scraping, including a URL and the traversal depth.
// It is used to define the specific web page to be scraped and its depth within the traversal.
type scrapeTarget struct {
	url   string
	depth int
}

// scrape is responsible for performing web scraping for a given target, fetching the page content,
// extracting data, and queuing new URLs with increased depth for further scraping. It also pipes the data into the analyzer.
func (s *Scrapper) scrape(ctx context.Context, id uint64, target scrapeTarget) error {
	// ctx cancelled, abort the scrape early
	if err := ctx.Err(); err != nil {
		return err
	}

	page, err := fetchPage(ctx, target.url, http.DefaultClient)
	if err != nil {
		fmt.Println("failed fetching page", "url", target.url, "err", err.Error())
		return err
	}
	fmt.Println("fetched new page", "size", len(page), "url", target.url)

	urls, words := s.extract(page)
	fmt.Println("extracted new data", "urls", len(urls), "words", len(words), "url", target.url)
	// queue all of the found urls with increased depth
	s.queueUrls(urls, target.depth+1)
	// analyze the words
	s.analyzer.Analyze(target.url, words)
	return nil
}

// extract extracts URLs and words from a given web page content.
func (s *Scrapper) extract(page string) (urls []string, words []string) {
	urlExtractor, wordExtractor := html.NewUrlsExtractor(), html.NewWordsExtractor()
	html.ExtractFromPage(page, []html.Extractor{urlExtractor, wordExtractor})
	urls = urlExtractor.Extracted()
	words = wordExtractor.Extracted()
	return
}

// queueUrls queues a list of URLs with increased depth for further scraping.
func (s *Scrapper) queueUrls(urls []string, currentDepth int) {
	if len(urls) == 0 {
		return
	}
	targets := make([]scrapeTarget, len(urls))
	for i, url := range urls {
		targets[i] = scrapeTarget{
			url:   url,
			depth: currentDepth,
		}
	}
	s.requestScrapes(targets)
}

// fetchPage fetches the content of a web page.
func fetchPage(ctx context.Context, url string, client *http.Client) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
