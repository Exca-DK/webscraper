package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Exca-DK/webscraper/scraper/html"
)

type scrapeTarget struct {
	url   string
	depth int
}

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

func (s *Scrapper) extract(page string) (urls []string, words []string) {
	urlExtractor, wordExtractor := html.NewUrlsExtractor(), html.NewWordsExtractor()
	html.ExtractFromPage(page, []html.Extractor{urlExtractor, wordExtractor})
	urls = urlExtractor.Extracted()
	words = wordExtractor.Extracted()
	return
}

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
