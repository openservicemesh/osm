package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"

	"github.com/openservicemesh/osm/demo/cmd/common"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	booksSold int64 = 0
	log             = logger.NewPretty("bookstore")
	identity        = flag.String("ident", "unidentified", "the identity of the container where this demo app is running (VM, K8s, etc)")
	port            = flag.Int("port", 80, "port on which this app is listening for incoming HTTP")
	path            = flag.String("path", ".", "path to the HTML template")
)

type handler struct {
	path   string
	fn     func(http.ResponseWriter, *http.Request)
	method string
}

func getIdentity() string {
	ident := os.Getenv("IDENTITY")
	if ident == "" {
		if identity != nil {
			ident = *identity
		}
	}
	return ident
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Set(common.BooksBoughtHeader, fmt.Sprintf("%d", booksSold))
	w.Header().Set(common.IdentityHeader, getIdentity())
}

func renderTemplate(w http.ResponseWriter) {
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/bookstore.html.template", *path))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse HTML template file")
	}
	err = tmpl.Execute(w, map[string]string{
		"Identity":  getIdentity(),
		"BooksSold": fmt.Sprintf("%d", booksSold),
		"Time":      time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Could not render template")
	}
}

func getBooksSold(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)
	renderTemplate(w)
	log.Info().Msgf("%s;  URL: %q;  Count: %d\n", getIdentity(), html.EscapeString(r.URL.Path), booksSold)
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)
	renderTemplate(w)
	log.Info().Msgf("%s;  URL: %q;  Count: %d\n", getIdentity(), html.EscapeString(r.URL.Path), booksSold)
}

// updateBooksSold updates the booksSold value to the one specified by the user
func updateBooksSold(w http.ResponseWriter, r *http.Request) {
	var updatedBooksSold int64
	err := json.NewDecoder(r.Body).Decode(&updatedBooksSold)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not decode request body")
	}
	atomic.StoreInt64(&booksSold, updatedBooksSold)
	setHeaders(w)
	renderTemplate(w)
	log.Info().Msgf("%s;  URL: %q;  %s: %d\n", getIdentity(), html.EscapeString(r.URL.Path), common.BooksBoughtHeader, booksSold)
}

// sellBook increments the value of the booksSold
func sellBook(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Selling a book!")
	atomic.AddInt64(&booksSold, 1)
	setHeaders(w)
	renderTemplate(w)
	log.Info().Msgf("%s;  URL: %q;  Count: %d\n", getIdentity(), html.EscapeString(r.URL.Path), booksSold)
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			log.Info().Msgf("%v: %v", name, h)
		}
	}

	go common.RestockBooks(1) // make this async for a smoother demo

	// Slow down the responses artificially.
	maxNoiseMilliseconds := 750
	minNoiseMilliseconds := 150
	intNoise := rand.Intn(maxNoiseMilliseconds-minNoiseMilliseconds) + minNoiseMilliseconds // #nosec G404
	pretendToBeBusy := time.Duration(intNoise) * time.Millisecond
	log.Info().Msgf("Sleeping %+v", pretendToBeBusy)
	time.Sleep(pretendToBeBusy)
}

func getHandlers() []handler {
	return []handler{
		{"/", getIndex, "GET"},
		{"/books-bought", getBooksSold, "GET"},
		{"/books-bought", updateBooksSold, "POST"},
		{"/buy-a-book/new", sellBook, "GET"},
		{"/reset", reset, "GET"},
		{"/liveness", ok, "GET"},
		{"/readiness", ok, "GET"},
		{"/startup", ok, "GET"},
	}
}

func reset(w http.ResponseWriter, _ *http.Request) {
	booksSold = 0
	renderTemplate(w)
}

func ok(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w)
}

func main() {
	flag.Parse()

	router := mux.NewRouter()

	for _, h := range getHandlers() {
		router.HandleFunc(h.path, h.fn).Methods(h.method)
	}
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})

	log.Info().Msgf("Bookstore running on port %d", *port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), router)
	log.Fatal().Err(err).Msgf("Failed to start HTTP server on port %d", *port)
}
