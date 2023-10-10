package html

import "golang.org/x/net/html"

// extractor is an interface that defines the extraction behavior for processing HTML content.
// Implementing types should define the 'extract' method to perform specific extraction actions
// based on the HTML structure and tokens.
type extractor interface {
	extract(tokenizer *html.Tokenizer, previous, current html.Token)
	Extracted() []string
}

type extractorData struct {
	extracted []string
}

func (e *extractorData) Extracted() []string { return e.extracted }

type wordExtractor struct {
	extractorData
}

func NewWordsExtractor() extractor {
	return &wordExtractor{
		extractorData: extractorData{
			extracted: make([]string, 0),
		},
	}
}

func (w *wordExtractor) extract(tokenizer *html.Tokenizer, previous, current html.Token) {
	w.extracted = append(w.extracted, extractWords(previous, current)...)
}

type urlsExtractor struct {
	extractorData
}

func NewUrlsExtractor() extractor {
	return &urlsExtractor{
		extractorData: extractorData{
			extracted: make([]string, 0),
		},
	}
}

func (w *urlsExtractor) extract(tokenizer *html.Tokenizer, previous, current html.Token) {
	w.extracted = append(w.extracted, extractUrlsFromToken(current)...)
}