//
// Hacky solution incoming
//

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-redis/redis_rate"
	"github.com/google/subcommands"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/kyokomi/emoji"
	"github.com/microcosm-cc/bluemonday"
	"github.com/shurcooL/github_flavored_markdown"
)

//
// Session secret
//
var store *sessions.CookieStore

//
// Rate-limiter
//
var rateLimiter *redis_rate.Limiter

// RemoteIP retrieves the remote IP of the request-submitter, taking
// account of any X-Forwarded-For header that might be submitted.
func RemoteIP(request *http.Request) string {

	//
	// Get the X-Forwarded-For header, if present.
	//
	xForwardedFor := request.Header.Get("X-Forwarded-For")

	//
	// No forwarded IP?  Then use the remote address directly.
	//
	if xForwardedFor == "" {
		ip, _, _ := net.SplitHostPort(request.RemoteAddr)
		return ip
	}

	entries := strings.Split(xForwardedFor, ",")
	address := strings.TrimSpace(entries[0])
	return (address)
}

// AddRateLimiting is a wrapper placed around all of our HTTP-methods, such
// that rate-limiting will be invoked.
func AddRateLimiting(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if rateLimiter == nil {
			next.ServeHTTP(w, r)
			return
		}

		//
		// Lookup the remote IP and limit to 200/Hour
		//
		ip := RemoteIP(r)
		limit := int64(200)

		//
		// If we've got a rate-limiter then we can use it.
		//
		rate, delay, allowed := rateLimiter.AllowHour(ip, limit)

		h := w.Header()

		//
		// We'll return the rate-limit headers to the caller.
		//
		h.Set("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
		h.Set("X-RateLimit-IP", ip)
		h.Set("X-RateLimit-Remaining", strconv.FormatInt(limit-rate, 10))
		delaySec := int64(delay / time.Second)
		h.Set("X-RateLimit-Delay", strconv.FormatInt(delaySec, 10))

		//
		// If the limit has been exceeded tell the client.
		//
		if !allowed {
			http.Error(w, fmt.Sprintf("API rate limit exceeded %d/hour.", limit), 429)
			return
		}

		//
		// Otherwise invoke the wrapped-handler.
		//
		next.ServeHTTP(w, r)
	})
}

// Render receives markdown, and returns (safe) HTML.
func Render(markdown string) string {

	// Convert the markdown -> html
	unsafe := github_flavored_markdown.Markdown([]byte(markdown))

	// Escape XSS, etc.
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)

	// This is a bit horrid - but expand the emoji
	emoji := emoji.Sprint(string(html))

	// And return our rendered output.
	return (emoji)
}

// PathHandler serves from our embedded resource(s)
func PathHandler(res http.ResponseWriter, req *http.Request) {

	//
	// Get the path, and handle "/" -> "/index.html"
	//
	path := req.URL.Path
	if strings.HasSuffix(path, "/") {
		path += "index.html"
	}

	//
	// Serve from our static-contents
	//
	data, err := ExpandResource("data/static" + path)
	if err != nil {
		fmt.Fprintf(res, err.Error())
		return
	}

	// Content-Type
	if strings.Contains(path, "html") {
		res.Header().Set("Content-Type", "text/html")
	}
	if strings.Contains(path, ".css") {
		res.Header().Set("Content-Type", "text/css")
	}
	if strings.Contains(path, ".js") {
		res.Header().Set("Content-Type", "text/javascript")
	}
	if strings.Contains(path, ".ico") {
		res.Header().Set("Content-Type", "image/x-icon")
	}
	if strings.Contains(path, ".txt") {
		res.Header().Set("Content-Type", "text/plain")
	}

	// Send the output back.
	fmt.Fprintf(res, "%s", data)

}

// CreateMarkdownHandler is the API end-point the user hits to create a
// new entry
func CreateMarkdownHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)
		}
	}()

	//
	// Get our input, if any
	//
	req.ParseForm()
	content := strings.Join(req.Form["text"], "")
	submit := strings.Join(req.Form["submit"], "")

	//
	// The data we add to our output-page
	//
	type Pagedata struct {
		HTML    string
		Content string
	}

	//
	// Populate our output page.
	//
	var x Pagedata
	x.HTML = Render(content)
	x.Content = content

	//
	// Load our template resource.
	//
	tmpl, err := ExpandResource("data/templates/create.tmpl")
	if err != nil {
		status = http.StatusNotFound
		return
	}

	//
	//  Load our template, from the resource.
	//
	src := string(tmpl)
	t := template.Must(template.New("tmpl").Parse(src))

	//
	// Now at this point we've got our rendered
	// HTML - which works for a preview - but if the
	// user wanted to save then we should do that
	// instead.
	//
	if submit == "Create" && len(content) > 0 {

		//
		// Add an entry
		//
		ip := RemoteIP(req)

		var key string
		var auth string

		key, auth, err = SaveMarkdown(content, ip)
		if err != nil {
			status = http.StatusNotFound
			return
		}

		//
		// Save the data.
		//
		if store != nil {
			session, _ := store.Get(req, "session-name")
			session.Values["auth"] = auth
			session.Save(req, res)
		}

		// Now redirect to view
		//
		http.Redirect(res, req, "/view/"+key, 302)
		return

	}

	//
	// If the user wants to use us programmatically
	//
	accept := req.FormValue("accept")
	if len(accept) < 1 {
		accept = req.Header.Get("Accept")
	}

	switch accept {
	case "application/json":
		if len(content) < 1 {
			res.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(res, "Empty content")
			return
		}

		ip := RemoteIP(req)

		var key string
		var auth string
		key, auth, err = SaveMarkdown(content, ip)
		if err != nil {
			status = http.StatusNotFound
			return
		}

		//
		// We'll return some JSON to the caller.
		//
		tmp := make(map[string]string)
		tmp["id"] = key
		tmp["auth"] = auth
		tmp["link"] = "https://" + req.Host + "/view/" + key
		tmp["raw"] = "https://" + req.Host + "/raw/" + key
		tmp["delete"] = "https://" + req.Host + "/delete/" + auth
		tmp["edit"] = "https://" + req.Host + "/edit/" + auth
		out, _ := json.MarshalIndent(tmp, "", "     ")

		//
		// Serve it appropriately
		//
		res.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(res, "%s", out)
		return
	}

	//
	// Execute the template into our buffer.
	//
	buf := &bytes.Buffer{}
	err = t.Execute(buf, x)

	//
	// If there were errors, then show them.
	//
	if err != nil {
		status = http.StatusNotFound
		return
	}

	//
	// Otherwise write the result.
	//
	res.Header().Set("Content-Type", "text/html")
	buf.WriteTo(res)

}

// EditMarkdownHandler allows a user to edit/update their markdown.
func EditMarkdownHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)
		}
	}()

	//
	// Get our input, if any
	//
	req.ParseForm()
	content := strings.Join(req.Form["text"], "")
	submit := strings.Join(req.Form["submit"], "")

	//
	// Get the authentication-token
	//
	vars := mux.Vars(req)
	id := vars["id"]

	//
	// Ensure we received a parameter.
	//
	if len(id) < 1 {
		status = http.StatusNotFound
		err = errors.New("Missing 'id' parameter")
		return
	}

	//
	// Ensure the ID is something sensible.
	//
	reg, _ := regexp.Compile("^([0-9a-z-]+)$")
	if !reg.MatchString(id) {
		status = http.StatusInternalServerError
		err = errors.New("The authentication-token didn't pass our validation rule")
		return
	}

	//
	// Lookup the value
	//
	var key string
	key, err = KeyFromAuth(id)
	if err != nil {
		status = http.StatusNotFound
		err = errors.New("Invalid authentication-token.")
		return
	}
	if key == "" {
		status = http.StatusNotFound
		err = errors.New("The authentication-token was invalid.")
		return
	}

	if content == "" {
		content = getMarkdown(key)
	}

	//
	// The data we add to our output-page
	//
	type Pagedata struct {
		HTML    string
		Content string
		Key     string
	}

	//
	// Populate our output page.
	//
	var x Pagedata
	x.HTML = Render(content)
	x.Content = content
	x.Key = id

	//
	// Load our template resource.
	//
	var tmpl string
	tmpl, err = ExpandResource("data/templates/edit.tmpl")
	if err != nil {
		status = http.StatusNotFound
		return
	}

	//
	//  Load our template, from the resource.
	//
	src := string(tmpl)
	t := template.Must(template.New("tmpl").Parse(src))

	//
	// Now at this point we've got our rendered
	// HTML - which works for a preview - but if the
	// user wanted to save then we should do that
	// instead.
	//
	if submit == "Update" && len(content) > 0 {

		//
		// Update the markdown
		//
		err = UpdateMarkdown(key, content)
		if err != nil {
			return
		}

		//
		// Redirect to view
		//
		http.Redirect(res, req, "/view/"+key, 302)
		return
	}

	//
	// Execute the template into our buffer.
	//
	buf := &bytes.Buffer{}
	err = t.Execute(buf, x)

	//
	// If there were errors, then show them.
	//
	if err != nil {
		fmt.Fprintf(res, err.Error())
		return
	}

	//
	// Otherwise write the result to the caller.
	//
	res.Header().Set("Content-Type", "text/html")
	buf.WriteTo(res)

}

// DeleteMarkdownHandler removes an entry.
func DeleteMarkdownHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)
		}
	}()

	//
	// Get the authentication-token
	//
	vars := mux.Vars(req)
	id := vars["id"]

	//
	// Ensure we received a parameter.
	//
	if len(id) < 1 {
		status = http.StatusNotFound
		err = errors.New("Missing 'id' parameter")
		return
	}

	//
	// Ensure the ID is something sensible.
	//
	reg, _ := regexp.Compile("^([0-9a-z-]+)$")
	if !reg.MatchString(id) {
		status = http.StatusInternalServerError
		err = errors.New("The authentication-token didn't pass our validation rule")
		return
	}

	//
	// Delete the entry.
	//
	err = DeleteMarkdown(id)
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// Now redirect to the site root
	//
	http.Redirect(res, req, "/", 302)
	return
}

// ViewMarkdownHandler fetches the text from our database, converts it
// to HTML and renders it.
//
// This is the core of our application.  See ViewRawMarkdownHandler to
// just display the raw version of the markdown (i.e. non-rendered)
//
func ViewMarkdownHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)
		}
	}()

	//
	// Get the ID of the thing we're going to view.
	//
	vars := mux.Vars(req)
	id := vars["id"]

	//
	// Ensure we received a parameter.
	//
	if len(id) < 1 {
		status = http.StatusNotFound
		err = errors.New("Missing 'id' parameter")
		return
	}

	//
	// Ensure the ID is something sensible.
	//
	reg, _ := regexp.Compile("^([0-9a-z-]+)$")
	if !reg.MatchString(id) {
		status = http.StatusInternalServerError
		err = errors.New("The markdown ID didn't pass our validation rule")
		return
	}

	//
	// Get the content.
	//
	content := getMarkdown(id)

	//
	// If that failed then look for an embedded markdown-resource instead.
	//
	if len(content) == 0 {
		var tmpl []byte
		tmpl, err = getResource("data/markdown/" + id + ".md")
		if err != nil {
			status = http.StatusNotFound
			err = errors.New("Markdown wasn't found for the given id")
			return
		}
		content = string(tmpl)
	}

	//
	// The data we add to our output-page
	//
	type Pagedata struct {
		ID   string
		HTML string
		Auth string
	}

	//
	// Populate.
	//
	var x Pagedata
	x.ID = id
	x.HTML = Render(content)
	x.Auth = ""

	//
	// Get the auth-value, if present, for the single time it
	// will be shown.
	//
	if store != nil {
		session, _ := store.Get(req, "session-name")
		auth := session.Values["auth"]
		if auth != nil {
			x.Auth = auth.(string)
			session.Values["auth"] = ""
			session.Save(req, res)
		}
	}

	//
	// Special-case:
	//
	//   /view/xxx -> Shows wrapped markdown in HTML
	//
	//   /html/xxx -> Shows unwrapped markdown in HTML
	//
	var tmpl string

	//
	// We either show wrapped, or unwrapped.
	//
	if strings.HasPrefix(req.URL.Path, "/html") {
		tmpl, err = ExpandResource("data/templates/view_raw.tmpl")
	} else {
		tmpl, err = ExpandResource("data/templates/view.tmpl")
	}

	//
	// If the template-loading failed we're in trouble.
	//
	if err != nil {
		status = http.StatusNotFound
		return
	}

	//
	//  Compile the template.
	//
	src := string(tmpl)
	t := template.Must(template.New("tmpl").Parse(src))

	//
	// Execute the template into our buffer.
	//
	buf := &bytes.Buffer{}
	err = t.Execute(buf, x)

	//
	// If there were errors, then show them.
	//
	if err != nil {
		status = http.StatusNotFound
		return
	}

	//
	// Otherwise write the result.
	//
	res.Header().Set("Content-Type", "text/html")
	buf.WriteTo(res)
}

// ViewRawMarkdownHandler returns the text the user initially added,
// without rendering it to HTML.
func ViewRawMarkdownHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)
		}
	}()

	//
	// Get the ID the user wants to view.
	//
	vars := mux.Vars(req)
	id := vars["id"]

	//
	// Ensure we received a parameter.
	//
	if len(id) < 1 {
		status = http.StatusNotFound
		err = errors.New("Missing 'id' parameter")
		return
	}

	//
	// Ensure the ID is something sensible.
	//
	reg, _ := regexp.Compile("^([0-9a-z-]+)$")
	if !reg.MatchString(id) {
		status = http.StatusInternalServerError
		err = errors.New("The markdown ID didn't pass our validation rule")
		return
	}

	//
	// Get the content.
	//
	content := getMarkdown(id)

	//
	// If that failed then look for an embedded markdown-resource instead.
	//
	if len(content) == 0 {
		var tmpl []byte
		tmpl, err = getResource("data/markdown/" + id + ".md")
		if err != nil {
			status = http.StatusNotFound
			err = errors.New("Markdown wasn't found for the given id")
			return
		}
		content = string(tmpl)
	}

	//
	// Load our template resource.
	//
	tmpl, err := ExpandResource("data/templates/raw.tmpl")
	if err != nil {
		status = http.StatusNotFound
		return
	}

	//
	// Compile the template
	//
	src := string(tmpl)
	t := template.Must(template.New("tmpl").Parse(src))

	//
	// Execute the template into our buffer.
	//
	buf := &bytes.Buffer{}
	err = t.Execute(buf, content)

	//
	// If there were errors, then show them.
	//
	if err != nil {
		status = http.StatusNotFound
		return
	}

	//
	// Otherwise write the result.
	//
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	buf.WriteTo(res)
}

//
// The options set by our command-line flags.
//
type serveCmd struct {
	//
	// The host to bind our server upon
	//
	bindHost string

	//
	// The port to listen upon
	//
	bindPort int

	//
	// The (optional) redis-host  to use for rate-limiting
	//
	redisHost string
}

//
// Glue
//
func (*serveCmd) Name() string     { return "serve" }
func (*serveCmd) Synopsis() string { return "Launch the HTTP server." }
func (*serveCmd) Usage() string {
	return `serve [options]:
  Launch the HTTP server for receiving reports & viewing them
`
}

//
// Flag setup
//
func (p *serveCmd) SetFlags(f *flag.FlagSet) {
	f.IntVar(&p.bindPort, "port", 3737, "The port to bind upon.")
	f.StringVar(&p.bindHost, "host", "127.0.0.1", "The IP to listen upon.")
	f.StringVar(&p.redisHost, "redis", "", "The address and port of a redis-server for rate-limiting.")
}

//
// Entry-point.
//
func (p *serveCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	//
	// If we have a redis-server specified, then use it.
	//
	if p.redisHost != "" {
		ring := redis.NewRing(&redis.RingOptions{
			Addrs: map[string]string{
				"server1": p.redisHost,
			},
		})

		rateLimiter = redis_rate.NewLimiter(ring)
	}

	//
	// Populate our cookie store with a random
	// authentication + encryption secret for security.
	//
	auth := make([]byte, 32)
	enc := make([]byte, 32)
	rand.Read(auth)
	rand.Read(enc)
	store = sessions.NewCookieStore(auth, enc)

	//
	// Create a new router and our route-mappings.
	//
	router := mux.NewRouter()

	//
	// Create.
	//
	//
	router.HandleFunc("/create/", CreateMarkdownHandler).Methods("GET")
	router.HandleFunc("/create", CreateMarkdownHandler).Methods("GET")
	router.HandleFunc("/create/", CreateMarkdownHandler).Methods("POST")
	router.HandleFunc("/create", CreateMarkdownHandler).Methods("POST")

	//
	// Edit.
	//
	router.HandleFunc("/edit/{id}", EditMarkdownHandler).Methods("GET")
	router.HandleFunc("/edit/{id}", EditMarkdownHandler).Methods("POST")
	router.HandleFunc("/edit/{id}/", EditMarkdownHandler).Methods("GET")
	router.HandleFunc("/edit/{id}/", EditMarkdownHandler).Methods("POST")

	//
	// Delete.
	//
	router.HandleFunc("/delete/{id}/", DeleteMarkdownHandler).Methods("GET")
	router.HandleFunc("/delete/{id}", DeleteMarkdownHandler).Methods("GET")

	//
	// View.
	//
	router.HandleFunc("/view/{id}/", ViewMarkdownHandler).Methods("GET")
	router.HandleFunc("/view/{id}", ViewMarkdownHandler).Methods("GET")
	router.HandleFunc("/html/{id}/", ViewMarkdownHandler).Methods("GET")
	router.HandleFunc("/html/{id}", ViewMarkdownHandler).Methods("GET")

	//
	// Raw.
	//
	router.HandleFunc("/raw/{id}/", ViewRawMarkdownHandler).Methods("GET")
	router.HandleFunc("/raw/{id}", ViewRawMarkdownHandler).Methods("GET")

	//
	// Static files - index.html, robots.txt, etc.
	//
	router.NotFoundHandler = http.HandlerFunc(PathHandler)

	//
	// Bind the router.
	//
	http.Handle("/", router)

	//
	// Show where we'll bind
	//
	bind := fmt.Sprintf("%s:%d", "127.0.0.1", 3737)
	fmt.Printf("Launching the server on http://%s\n", bind)

	//
	// Wire up logging.
	//
	loggedRouter := handlers.LoggingHandler(os.Stdout, router)

	//
	// Wire up context (i.e. rate-limiter)
	//
	contextRouter := AddRateLimiting(loggedRouter)

	//
	// Launch the server.
	//
	err := http.ListenAndServe(bind, contextRouter)
	if err != nil {
		fmt.Printf("\nError launching server: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	//
	// All done.
	//
	return subcommands.ExitSuccess
}
