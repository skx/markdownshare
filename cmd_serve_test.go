//
// Simple testing of the HTTP-server
//
//
package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
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
	router.HandleFunc("/raw/{id}/", ViewRawMarkdownHandler).Methods("GET")
	router.HandleFunc("/raw/{id}", ViewRawMarkdownHandler).Methods("GET")

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
		{"/raw/konami", "the [Konami Code](http://en.wikipedia"},
		{"/raw/konami/", "the [Konami Code](http://en.wikipedia"},
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
func TestMissingID(t *testing.T) {

	router := mux.NewRouter()
	router.HandleFunc("/delete/", DeleteMarkdownHandler).Methods("GET")
	router.HandleFunc("/html/", ViewMarkdownHandler).Methods("GET")
	router.HandleFunc("/raw/", ViewRawMarkdownHandler).Methods("GET")
	router.HandleFunc("/view/", ViewMarkdownHandler).Methods("GET")

	// Some bogus IDS
	tests := []string{
		"/delete/",
		"/html/",
		"/raw/",
		"/view/",
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

// Test invalid IDs
func TestBogusID(t *testing.T) {

	router := mux.NewRouter()
	router.HandleFunc("/delete/{id}", DeleteMarkdownHandler).Methods("GET")
	router.HandleFunc("/html/{id}", ViewMarkdownHandler).Methods("GET")
	router.HandleFunc("/raw/{id}", ViewRawMarkdownHandler).Methods("GET")
	router.HandleFunc("/view/{id}", ViewMarkdownHandler).Methods("GET")

	// Some bogus IDS
	tests := []string{
		"/delete/$(id)",
		"/html/`uptime`",
		"/raw/;",
		"/view/xx:yy",
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
		if !strings.Contains(rr.Body.String(), "validation rule") {
			t.Errorf("handler returned unexpected body -  %s",
				rr.Body.String())
		}
	}

}

// This function tries to test uploading a piece of text, to test
// the preview - but not the saving - of markdown.
func TestPreview(t *testing.T) {

	//
	// Create a temporary directory to store uploads
	//
	p, err := ioutil.TempDir(os.TempDir(), "prefix")
	if err == nil {
		PREFIX = p + "/"
	} else {
		t.Fatal(err)
	}

	//
	// We're going to post some text, but crucially not the
	// `submit` parameter.
	//
	// That means we'll be testing the preview-behaviour, rather
	// than the create behavior.
	//
	data := url.Values{}
	data.Set("text", "__bold__")
	data.Set("submit", "Preview")

	req, err := http.NewRequest("POST", "/create", bytes.NewBufferString(data.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	//
	// Record via the handler.
	//
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateMarkdownHandler)

	// Our handlers satisfy http.Handler, so we can call
	// their ServeHTTP method directly and pass in our
	// Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Unexpected status-code: %v", status)
	}

	// Check the response body contains the rendered markdown.
	if !strings.Contains(rr.Body.String(), "<p><strong>bold</strong></p>") {
		t.Errorf("handler returned unexpected body: got '%v'",
			rr.Body.String())
	}

	//
	// Cleanup our temporary directory
	//
	os.RemoveAll(p)
}

// Test uploading markdown via the API-method, which should return JSON.
//
// Note: We don't test what we got, just that we got "json".
//
func TestAPICreate(t *testing.T) {

	//
	// Create a temporary directory to store uploads
	//
	p, err := ioutil.TempDir(os.TempDir(), "apiupload")
	if err == nil {
		PREFIX = p + "/"
	} else {
		t.Fatal(err)
	}

	data := url.Values{}
	data.Set("text", "__API__ upload!")
	data.Set("accept", "application/json")

	req, err := http.NewRequest("POST", "/create", bytes.NewBufferString(data.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	//
	// Record via the handler.
	//
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateMarkdownHandler)

	// Our handlers satisfy http.Handler, so we can call
	// their ServeHTTP method directly and pass in our
	// Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Unexpected status-code: %v", status)
	}

	// Check the response body contains the rendered markdown we
	// submitted.
	if !strings.Contains(rr.Body.String(), "\"link\":") {
		t.Errorf("handler returned unexpected body: got '%v'",
			rr.Body.String())
	}

	if ctype := rr.Header().Get("Content-Type"); ctype != "application/json" {
		t.Errorf("Content-Type was not JSON: %s", ctype)
	}

	//
	// Cleanup our temporary directory
	//
	os.RemoveAll(p)
}

// Test creating & fetching markdown.
func TestCreateAndView(t *testing.T) {

	//
	// Create a temporary directory to store uploads
	//
	p, err := ioutil.TempDir(os.TempDir(), "apiupload")
	if err == nil {
		PREFIX = p + "/"
	} else {
		t.Fatal(err)
	}

	data := url.Values{}
	data.Set("text", "[steve.fi](https://steve.fi/)")
	data.Set("submit", "Create")

	req, err := http.NewRequest("POST", "/create", bytes.NewBufferString(data.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	//
	// Record via the handler.
	//
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateMarkdownHandler)

	// Our handlers satisfy http.Handler, so we can call
	// their ServeHTTP method directly and pass in our
	// Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != 302 {
		t.Errorf("Unexpected status-code: %v", status)
	}

	// Check the redirection target starts with /view/
	target := rr.HeaderMap.Get("Location")
	if !strings.HasPrefix(target, "/view") {
		t.Errorf("Redirection target looks bogus")
	}

	// OK now try to get that markdown - via the lookup not
	// a HTTP-fetch right now.
	target = strings.TrimPrefix(target, "/view/")
	markdown := getMarkdown(target)

	//
	// Should have raw markdown.
	//
	if !strings.Contains(markdown, "(https://steve.fi/)") {
		t.Errorf("Markdown didn't look correct: %s\n", markdown)
	}

	//
	// Secondly try to fetch via the HTTP-handler
	//
	router := mux.NewRouter()
	router.HandleFunc("/view/{id}", ViewMarkdownHandler).Methods("GET")

	req, err = http.NewRequest("GET", "/view/"+target, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("Unexpected status-code: %v", status)
	}

	// Check the response body is what we expect.
	if !strings.Contains(rr.Body.String(), "href=\"https://steve.fi/\" rel=\"nofollow") {
		t.Errorf("handler returned unexpected body: %s", rr.Body.String())
	}

	//
	// Cleanup our temporary directory
	//
	os.RemoveAll(p)
}

// Test creating & deleting markdown.
func TestCreateAndDelete(t *testing.T) {

	//
	// Create a temporary directory to store uploads
	//
	p, err := ioutil.TempDir(os.TempDir(), "apiupload")
	if err == nil {
		PREFIX = p + "/"
	} else {
		t.Fatal(err)
	}

	data := url.Values{}
	data.Set("text", "[steve.fi](https://steve.fi/)")
	data.Set("submit", "Create")

	req, err := http.NewRequest("POST", "/create", bytes.NewBufferString(data.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	//
	// Record via the handler.
	//
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateMarkdownHandler)

	// Our handlers satisfy http.Handler, so we can call
	// their ServeHTTP method directly and pass in our
	// Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != 302 {
		t.Errorf("Unexpected status-code: %v", status)
	}

	// Check the redirection target starts with /view/
	target := rr.HeaderMap.Get("Location")
	if !strings.HasPrefix(target, "/view") {
		t.Errorf("Redirection target looks bogus")
	}

	// OK now try to get that markdown - via the lookup not
	// a HTTP-fetch right now.
	target = strings.TrimPrefix(target, "/view/")
	markdown := getMarkdown(target)

	//
	// Should have raw markdown.
	//
	if !strings.Contains(markdown, "(https://steve.fi/)") {
		t.Errorf("Markdown didn't look correct: %s\n", markdown)
	}

	//
	// Now we delete the markdown.
	//
	// We have to check to get the authentication-token, which
	// would have been set via a cookie.
	//
	token, _ := readFile(target + ".AUTH")

	//
	// Make the deletion request
	//
	router := mux.NewRouter()
	router.HandleFunc("/delete/{id}", DeleteMarkdownHandler).Methods("GET")
	req, err = http.NewRequest("GET", "/delete/"+token, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != 302 {
		t.Errorf("Unexpected status-code: %v", status)
	}

	//
	// At this point re-fetching the body should fail.
	//
	markdown = getMarkdown(target)
	if markdown != "" {
		t.Errorf("Expected deleted markdown to be empty - got %s\n", markdown)
	}

	//
	// Cleanup our temporary directory
	//
	os.RemoveAll(p)
}
