//
// Simple testing of the HTTP-server
//
//
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// Test fetching some static-resources.
//
func TestStaticResources(t *testing.T) {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(PathHandler)

	// Structure of requested-path and content we expected to find
	type TestCase struct {
		URL     string
		Content string
	}

	tests := []TestCase{
		{"/robots.txt", "Crawl-delay"},
		{"/humans.txt", "Kemp"},
		{"/index.html", "Simple Markdown sharing"},
		{"/", "Simple Markdown sharing"},
		{"/markdownshare.com.conf", "/create"},
	}

	for _, entry := range tests {
		req, err := http.NewRequest("GET", entry.URL, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Check the status code is what we expect.
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Unexpected status-code: %v", status)
		}

		// Check the response looks good.
		if !strings.Contains(rr.Body.String(), entry.Content) {
			t.Errorf("handler returned unexpected body: got '%v' want '%v'",
				rr.Body.String(), entry.Content)
		}
	}
}

func TestMarkdownRendering(t *testing.T) {

	type TestCase struct {
		Input  string
		Output string
	}

	tests := []TestCase{
		{"Test", "<p>Test</p>"},
		{"_italic_", "<p><em>italic</em></p>"},
		{"__bold__", "<p><strong>bold</strong></p>"},
		{"[Steve](https://steve.fi/)", "<p><a href=\"https://steve.fi/\" rel=\"nofollow\">Steve</a></p>"},
	}

	for _, entry := range tests {
		html := Render(entry.Input)
		html = strings.TrimSpace(html)

		if html+"" != entry.Output {
			t.Errorf("Markdown rendering gave wrong result - expected '%s' - got '%s'", entry.Output, html)
		}
	}
}
