package analytics

import (
	"sort"
	"strings"
	"sync"

	"github.com/Exca-DK/webscraper/scraper/html"
)

// Analyzer is an interface designed for analyzing various data from scraped page.
// Either Analyze or Cancel must be called in order to proper close the resource.
type Analyzer interface {
	// Analyze executes the arbitrary logic
	Analyze(page string)
	// Cancel ensures that analyzer won't be executed
	Cancel(err error)
}

// WordFrequencyAnalyzer is an object that counts words frequency in a page.
// The analyzer expects to be called only once!
type WordFrequencyAnalyzer struct {
	wg        sync.WaitGroup
	frequency map[string]uint
	err       error
}

// NewWordFrequencyAnalyzer() returns Analyzer that counts words frequency x times.
func NewWordFrequencyAnalyzer(times int) *WordFrequencyAnalyzer {
	analyzer := &WordFrequencyAnalyzer{
		frequency: make(map[string]uint),
	}
	analyzer.wg.Add(times)
	return analyzer
}

// Implements Analyzer.Analyze
func (analyzer *WordFrequencyAnalyzer) Analyze(page string) {
	for _, word := range html.ExtractWordsFromPage(page) {
		analyzer.frequency[strings.ToLower(word)]++
	}
	analyzer.wg.Done()
}

// Implements Analyzer.Cancel
func (analyzer *WordFrequencyAnalyzer) Cancel(err error) {
	analyzer.err = err
	analyzer.wg.Done()
}

// Result wait's for it's execution to be done and returns the result.
func (analyzer *WordFrequencyAnalyzer) Result() ([]struct {
	Word  string
	Count uint
}, error) {
	analyzer.wg.Wait()
	if analyzer.err != nil {
		return nil, analyzer.err
	}
	arr := make([]struct {
		Word  string
		Count uint
	}, 0, len(analyzer.frequency))
	for key, value := range analyzer.frequency {
		arr = append(arr, struct {
			Word  string
			Count uint
		}{
			Word:  key,
			Count: value,
		})
	}
	sort.SliceStable(arr, func(i, j int) bool {
		return arr[i].Count > arr[j].Count
	})
	return arr, nil
}
