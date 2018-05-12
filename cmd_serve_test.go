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
func TestStaticResources(t *testing.T) {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(PathHandler)

	// Structure of requested-path and content we expected to find
	type TestCase struct {
		URL     string
		Content string
		Type    string
	}

	tests := []TestCase{
		{"/robots.txt", "Crawl-delay", "text/plain"},
		{"/js/k.js", "KONAMI", "text/javascript"},
		{"/css/style.css", "font-family", "text/css"},
		{"/favicon.ico", "\xf2", "image/x-icon"},
		{"/humans.txt", "Kemp", "text/plain"},
		{"/index.html", "Simple Markdown sharing", "text/html"},
		{"/", "Simple Markdown sharing", "text/html"},
		{"/markdownshare.com.conf", "/create", "text/plain; charset=utf-8"},
		{"/resource/not/found", "Failed to find resource", "text/plain; charset=utf-8"},
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

		if ctype := rr.Header().Get("Content-Type"); ctype != entry.Type {
			t.Errorf("Content-Type header does not match %s: got %v want %v",
				entry.URL, ctype, entry.Type)
		}
		// Check the response looks good.
		if !strings.Contains(rr.Body.String(), entry.Content) {
			t.Errorf("handler returned unexpected body: got '%v' want '%v'",
				rr.Body.String(), entry.Content)
		}
	}
}

// Test (basic/naive) Markdown rendering.
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

// Test our konami-code is rendered
func TestKonamiCode(t *testing.T) {

	router := mux.NewRouter()
	router.HandleFunc("/view/{id}/", ViewMarkdownHandler).Methods("GET")
	router.HandleFunc("/view/{id}", ViewMarkdownHandler).Methods("GET")
	router.HandleFunc("/html/{id}/", ViewMarkdownHandler).Methods("GET")
	router.HandleFunc("/html/{id}", ViewMarkdownHandler).Methods("GET")

	type TestCase struct {
		URL    string
		Output string
	}

	tests := []TestCase{
		{"/view/konami", "konami"},
		{"/view/konami/", "konami"},
		{"/view/konami2/", "wasn't found for the given id"},
		{"/html/konami", "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"},
		{"/html/konami/", "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"},
	}

	for _, entry := range tests {
		req, err := http.NewRequest("GET", entry.URL, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Check the status code is what we expect.
		if status := rr.Code; status != http.StatusOK && status != http.StatusNotFound {
			t.Errorf("Unexpected status-code: %v", status)
		}

		// Check the response body is what we expect.
		if !strings.Contains(rr.Body.String(), entry.Output) {
			t.Errorf("handler returned unexpected body - didn't match %s: %s",
				entry.Output, rr.Body.String())
		}
	}

}

// Test ID validation is OK
func TestViewInvalidID(t *testing.T) {

	router := mux.NewRouter()
	router.HandleFunc("/view/{id}/", ViewMarkdownHandler).Methods("GET")
	router.HandleFunc("/view/{id}", ViewMarkdownHandler).Methods("GET")

	// Some bogus IDS
	tests := []string{
		"/view/$(id)",
		"/view/`uptime`/",
		"/view/lk<>?",
		"/view/ljK<>?/",
	}

	for _, uri := range tests {
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Check the status code is what we expect.
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Unexpected status-code: %v", status)
		}

		// Check the response body is what we expect.
		if !strings.Contains(rr.Body.String(), "pass our validation rule") {
			t.Errorf("handler returned unexpected body -  %s",
				rr.Body.String())
		}
	}

}

// Test ID is present
func TestViewMissingID(t *testing.T) {

	router := mux.NewRouter()
	router.HandleFunc("/view/", ViewMarkdownHandler).Methods("GET")
	router.HandleFunc("/html/", ViewMarkdownHandler).Methods("GET")

	// Some bogus IDS
	tests := []string{
		"/view/",
		"/html/",
	}

	for _, uri := range tests {
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Check the status code is what we expect.
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Unexpected status-code: %v", status)
		}

		// Check the response body is what we expect.
		if !strings.Contains(rr.Body.String(), "Missing 'id'") {
			t.Errorf("handler returned unexpected body -  %s",
				rr.Body.String())
		}
	}

}
