package scraper

import (
	"sync"
	"testing"
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

func TestScrape(t *testing.T) {
	analyzer := &testingAnalyzer{words: make([]string, 0)}
	scrapper := NewScrapper(analyzer).WithThreads(3).WithDepth(2)
	scrapper.Start()
	scrapper.Scrape("")
}
