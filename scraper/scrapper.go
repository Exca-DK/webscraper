package scraper

import (
	"io"
	"net/http"
)

type Scrapper struct {
	done  chan struct{}
	urlCh chan string
}

func (s *Scrapper) Scrape(url string) {
	select {
	case <-s.done:
	case s.urlCh <- url:
	}
}

func fetchPage(url string, client *http.Client) (string, error) {
	resp, err := client.Get(url)
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
