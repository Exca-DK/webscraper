package scraper

import (
	"context"
	"io"
	"net/http"

	"github.com/Exca-DK/webscraper/scraper/analytics"
)

// scrapeTarget represents a target for web scraping,.l.
type scrapeTarget struct {
	url      string
	analyzer analytics.Analyzer
}

// scrape is responsible for performing web scraping for a given target.
func (s *Scrapper) scrape(ctx context.Context, id uint64, target scrapeTarget) error {
	// ctx cancelled, abort the scrape early
	if err := ctx.Err(); err != nil {
		target.analyzer.Cancel(ctx.Err())
		return err
	}

	page, err := fetchPage(ctx, target.url, http.DefaultClient)
	if err != nil {
		target.analyzer.Cancel(err)
		return err
	}
	s.logger.Debug("fetched page", "jobId", id, "url", target.url, "size", len(page))
	target.analyzer.Analyze(page)
	return nil
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
