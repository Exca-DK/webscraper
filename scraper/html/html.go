package html

import (
	"net"
	"net/url"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/net/html"
)

var (
	readerPool = sync.Pool{
		New: func() any {
			return &strings.Reader{}
		},
	}
)

const (
	script = "script"
	css    = "style"
)

func getReader(text string) *strings.Reader {
	reader := readerPool.Get().(*strings.Reader)
	reader.Reset(text)
	return reader
}

func freeReader(r *strings.Reader) {
	readerPool.Put(r)
}

// ExtractWordsFromPage parses an HTML page represented as a string and extracts valid words.
// Valid words are extracted and returned as a slice of strings.
func ExtractWordsFromPage(page string) []string {
	reader := getReader(page)
	defer freeReader(reader)
	extr := NewWordsExtractor()
	Extract(html.NewTokenizer(reader), []Extractor{extr})
	return extr.Extracted()
}

// ExtractUrlsFromPage parses an HTML page represented as a string and extracts valid URLs.
// Valid URLs are extracted and returned as a slice of strings.
func ExtractUrlsFromPage(page string) []string {
	reader := getReader(page)
	defer freeReader(reader)
	extr := NewUrlsExtractor()
	Extract(html.NewTokenizer(reader), []Extractor{extr})
	return extr.Extracted()
}

// ExtractFromPage processes an HTML document as a string using a set of extractors.
func ExtractFromPage(page string, extractors []Extractor) {
	reader := getReader(page)
	defer freeReader(reader)
	Extract(html.NewTokenizer(reader), extractors)
}

// Extract processes an HTML document using an HTML tokenizer and a set of extractors.
// It iterates through the tokens in the HTML content, and for each token, it applies each extractor's
// extraction logic. Extractors are responsible for extracting specific information or performing actions
// based on the HTML structure.
func Extract(tokenizer *html.Tokenizer, extractors []Extractor) {
	previous := tokenizer.Token()
	for tokenType := tokenizer.Next(); tokenType != html.ErrorToken; tokenType = tokenizer.Next() {
		current := tokenizer.Token()
		for _, extractor := range extractors {
			extractor.extract(tokenizer, previous, current)
		}
		previous = current
	}
}

// extractWords extracts words from an HTML document using an HTML tokenizer.
// It iterates through the tokens in the HTML content and extracts words from text content.
// It takes the previous token type into account to properly identify and extract words.
func extractWords(previous html.Token, current html.Token) []string {
	if previous.Data == script || previous.Data == css {
		return nil
	}
	if current.Type != html.TextToken {
		return nil
	}

	sentence := strings.TrimSpace(html.UnescapeString(current.Data))
	if len(sentence) == 0 {
		return nil
	}
	return splitSentence(sentence)
}

func splitSentence(sentence string) []string {
	tmp := strings.Fields(sentence)
	result := make([]string, 0, len(tmp))
	for _, word := range tmp {
		sanitized, ok := sanitizeWord(word)
		if !ok {
			continue
		}
		result = append(result, sanitized)
	}
	return result
}

func sanitizeWord(word string) (string, bool) {
	// remove first non-asci char
	if !unicode.IsLetter(rune(word[0])) {
		word = word[0:]
	}
	// remove last non-asci char
	if !unicode.IsLetter(rune(word[len(word)-1])) {
		word = word[:len(word)-1]
	}
	for _, r := range word {
		if !unicode.IsLetter(r) {
			return "", false
		}
	}
	lowered := strings.ToLower(word)
	return lowered, len(lowered) != 0
}

// extractUrlsFromToken extracts valid URLs from token.
func extractUrlsFromToken(token html.Token) []string {
	if !(token.Data == "a") {
		return nil
	}
	href, ok := getHref(token)
	if !ok {
		return nil
	}
	return hrefToUrls(href)
}

// getHref extracts the value of the "href" attribute from an HTML token.
func getHref(token html.Token) (string, bool) {
	for _, a := range token.Attr {
		if a.Key == "href" {
			return a.Val, true
		}
	}
	return "", false
}

// hrefToUrls extracts valid URLs from the given string and returns them as a slice of strings.
func hrefToUrls(href string) []string {
	result := make([]string, 0)
	urls := strings.Split(href, "http")
	for _, url := range urls {
		url = "http" + url
		if !isValidUrl(url) {
			continue
		}
		result = append(result, url)
	}
	return result
}

// isValidUrl checks if the given string represents a valid URL with either the "http" or "https" scheme.
// It parses the input as a URL and verifies that it has a valid scheme ("http" or "https") and a valid host.
func isValidUrl(str string) bool {
	uri, err := url.ParseRequestURI(str)
	if err != nil {
		return false
	}

	// ensure scheme is valid
	if uri.Scheme != "http" && uri.Scheme != "https" {
		return false
	}

	host, _, err := net.SplitHostPort(uri.Host)
	if err != nil {
		host = uri.Host
	}

	// ensure that host is valid
	_, err = net.LookupHost(host)
	return err == nil
}
