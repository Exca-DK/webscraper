package html

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
)

func TestHrefToUrl(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	tests := []struct {
		raw  string
		want []string
	}{
		{
			raw:  "https://go.dev",
			want: []string{"https://go.dev"},
		},
		{
			raw:  "https://go.devhttp://go.dev",
			want: []string{"https://go.dev", "http://go.dev"},
		},
		{
			raw:  srv.URL,
			want: []string{srv.URL},
		},
	}

	for _, test := range tests {
		urls := hrefToUrls(test.raw)
		if len(urls) != len(test.want) {
			t.Fatalf("different result size. got %v want %v", urls, test.want)
		}
		for i := 0; i < len(urls); i++ {
			if urls[i] != test.want[i] {
				t.Fatalf("unexpected result. got %v want %v", urls, test.want)
			}
		}
	}
}

func TestExtraction(t *testing.T) {
	content, descr, err := loadTestData("./testdata", "google")
	if err != nil {
		t.Fatal(err)
	}
	verify := func(t *testing.T, expected, have []string) {
		if len(expected) != len(have) {
			t.Fatalf("unexpected words. got %v want %v", have, expected)
		}
		for i := 0; i < len(expected); i++ {
			if expected[i] != have[i] {
				t.Fatalf("unexpected words. got %s want %s", have[i], expected[i])
			}
		}
	}
	t.Run("single extractor", func(t *testing.T) {
		t.Run("words", func(t *testing.T) {
			verify(t, descr.Words, ExtractWordsFromPage(string(content)))
		})
		t.Run("urls", func(t *testing.T) {
			verify(t, descr.Urls, ExtractUrlsFromPage(string(content)))
		})
	})

	t.Run("multi extractor", func(t *testing.T) {
		wextr := NewWordsExtractor()
		uextr := NewUrlsExtractor()
		ExtractFromPage(string(content), []Extractor{wextr, uextr})
		verify(t, descr.Words, wextr.Extracted())
		verify(t, descr.Urls, uextr.Extracted())
	})
}

func TestUrlValidation(t *testing.T) {
	tests := []struct {
		raw      string
		expected bool
	}{
		{
			raw:      "https://go.dev",
			expected: true,
		},
		{
			raw:      "https//go.dev",
			expected: false,
		},
		{
			raw:      "http://godoc.org",
			expected: true,
		},
		{
			raw:      "http://godoc.org",
			expected: true,
		},
		{
			raw:      "godoc.org",
			expected: false,
		},
		{
			raw:      "godoc",
			expected: false,
		},
	}

	for _, test := range tests {
		valid := isValidUrl(test.raw)
		if valid != test.expected {
			t.Fatalf("unexpected test result. got %v want %v on %s\n", valid, test.expected, test.raw)
		}
	}
}

func writeTestData(content []byte, descr htmlTestDataDescription, dir string, name string) error {
	if !descr.Valid() {
		return errors.New("descr not valid")
	}
	descrData, err := descr.ToBytes()
	if err != nil {
		return err
	}
	err = writeToFile(content, fmt.Sprintf("%s.html", path.Join(dir, name)))
	if err != nil {
		return err
	}
	return writeToFile(descrData, fmt.Sprintf("%s.json", path.Join(dir, name)))
}

func loadTestData(dir string, name string) ([]byte, htmlTestDataDescription, error) {
	var descr htmlTestDataDescription
	data, err := os.ReadFile(fmt.Sprintf("%s.json", path.Join(dir, name)))
	if err != nil {
		return nil, descr, err
	}
	err = descr.FromBytes(data)
	if err != nil {
		return nil, descr, err
	}
	data, err = os.ReadFile(fmt.Sprintf("%s.html", path.Join(dir, name)))
	if err != nil {
		return nil, descr, err
	}
	return data, descr, err
}

func writeToFile(content []byte, filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(content)
	return err
}

type htmlTestDataDescription struct {
	Source string
	Words  []string
	Urls   []string
}

func (data htmlTestDataDescription) ToBytes() ([]byte, error) {
	return json.Marshal(data)
}

func (data *htmlTestDataDescription) FromBytes(b []byte) error {
	return json.Unmarshal(b, data)
}

func (data htmlTestDataDescription) Valid() bool {
	return len(data.Source) > 0 && len(data.Words) > 0 && len(data.Urls) > 0
}
