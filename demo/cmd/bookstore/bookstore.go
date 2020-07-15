package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/open-service-mesh/osm/demo/cmd/common"
	"github.com/open-service-mesh/osm/pkg/featureflags"
)

var identity = flag.String("ident", "unidentified", "the identity of the container where this demo app is running (VM, K8s, etc)")

var port = flag.Int("port", 8080, "port on which this app is listening for incoming HTTP")
var path = flag.String("path", ".", "path to the HTML template")
var booksBought int

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
	w.Header().Set(common.BooksBoughtHeader, fmt.Sprintf("%d", booksBought))
	w.Header().Set(common.IdentityHeader, getIdentity())
}

func renderTemplate(w http.ResponseWriter) {
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/bookstore.html.template", *path))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse HTML template file")
	}
	err = tmpl.Execute(w, map[string]string{
		common.IdentityHeader:    getIdentity(),
		common.BooksBoughtHeader: fmt.Sprintf("%d", booksBought),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Could not render template")
	}
}

func getBooksBought(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)
	renderTemplate(w)
	fmt.Printf("%s;  URL: %q;  Count: %d\n", getIdentity(), html.EscapeString(r.URL.Path), booksBought)
}

// updateBooksBought updates the booksBought value to the one specified by the user
func updateBooksBought(w http.ResponseWriter, r *http.Request) {
	var updatedBooksBought int
	err := json.NewDecoder(r.Body).Decode(&updatedBooksBought)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not decode request body")
	}
	booksBought = updatedBooksBought
	setHeaders(w)
	renderTemplate(w)
	fmt.Printf("%s;  URL: %q;  %s: %d\n", getIdentity(), html.EscapeString(r.URL.Path), common.BooksBoughtHeader, booksBought)
}

// buyBook increments the value of the booksBought
func buyBook(w http.ResponseWriter, r *http.Request) {
	booksBought++
	setHeaders(w)
	renderTemplate(w)
	fmt.Printf("%s;  URL: %q;  Count: %d\n", getIdentity(), html.EscapeString(r.URL.Path), booksBought)
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			fmt.Printf("%v: %v", name, h)
		}
	}

	common.RestockBooks(1)
}

func main() {
	flag.Parse()

	featureflags.Initialize(featureflags.OptionalFeatures{
		EnableHumanReadableLog: true,
	})

	booksBought = 1

	//initializing router
	router := mux.NewRouter()

	//endpoints
	router.HandleFunc("/books-bought", getBooksBought).Methods("GET")
	router.HandleFunc("/books-bought", updateBooksBought).Methods("POST")
	router.HandleFunc("/buy-a-book/new", buyBook).Methods("GET")
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), router)
	log.Fatal().Err(err).Msgf("Failed to start HTTP server on port %d", *port)
}
