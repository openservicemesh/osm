package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/open-service-mesh/osm/demo/cmd/common"

	"github.com/golang/glog"

	"github.com/gorilla/mux"
)

var identity = flag.String("ident", "unidentified", "the identity of the container where this demo app is running (VM, K8s, etc)")

var port = flag.Int("port", 80, "port on which this app is listening for incoming HTTP")
var path = flag.String("path", ".", "path to the HTML template")
var booksBought int
var tmpl *template.Template

func getIdentity() string {
	ident := os.Getenv("IDENTITY")
	if ident == "" {
		if identity != nil {
			ident = *identity
		}
	}
	return ident
}

// getCurrentBooksBought gets the value of the counter
func getCurrentBooksBought(w http.ResponseWriter, r *http.Request) {

	w.Header().Set(common.BooksBoughtHeader, fmt.Sprintf("%d", booksBought))

	w.Header().Set(common.IdentityHeader, getIdentity())
	err := tmpl.Execute(w, map[string]string{
		common.IdentityHeader:    getIdentity(),
		common.BooksBoughtHeader: fmt.Sprintf("%d", booksBought),
	})
	if err != nil {
		glog.Fatal("Could not render template", err)
	}
	fmt.Printf("%s;  URL: %q;  Count: %d\n", getIdentity(), html.EscapeString(r.URL.Path), booksBought)
}

// updateBooksBoughtValue updates the booksBought value to the one specified by the user
func updateBooksBoughtValue(w http.ResponseWriter, r *http.Request) {

	var updatedBooksBought int
	err := json.NewDecoder(r.Body).Decode(&updatedBooksBought)
	if err != nil {
		glog.Fatal("Could not decode request body", err)
	}
	booksBought = updatedBooksBought
	w.Header().Set(common.BooksBoughtHeader, fmt.Sprintf("%d", booksBought))
	w.Header().Set(common.IdentityHeader, getIdentity())
	err = tmpl.Execute(w, map[string]string{
		common.IdentityHeader:    getIdentity(),
		common.BooksBoughtHeader: fmt.Sprintf("%d", booksBought),
	})
	if err != nil {
		glog.Fatal("Could not render template", err)
	}
	fmt.Printf("%s;  URL: %q;  %s: %d\n", getIdentity(), html.EscapeString(r.URL.Path), common.BooksBoughtHeader, booksBought)
}

// incrementBooksBought increments the value of the booksBought
func incrementBooksBought(w http.ResponseWriter, r *http.Request) {

	booksBought++
	w.Header().Set(common.BooksBoughtHeader, fmt.Sprintf("%d", booksBought))
	w.Header().Set(common.IdentityHeader, getIdentity())
	err := tmpl.Execute(w, map[string]string{
		common.IdentityHeader:    getIdentity(),
		common.BooksBoughtHeader: fmt.Sprintf("%d", booksBought),
	})
	if err != nil {
		glog.Fatal("Could not render template", err)
	}
	fmt.Printf("%s;  URL: %q;  Count: %d\n", getIdentity(), html.EscapeString(r.URL.Path), booksBought)
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			fmt.Printf("%v: %v", name, h)
		}
	}
}

func main() {
	flag.Parse()
	var err error
	tmpl, err = template.ParseFiles(fmt.Sprintf("%s/bookstore.html.template", *path))
	if err != nil {
		log.Fatal(err)
	}
	booksBought = 1

	//initializing router
	router := mux.NewRouter()

	//endpoints
	router.HandleFunc("/books-bought", getCurrentBooksBought).Methods("GET")
	router.HandleFunc("/books-bought", updateBooksBoughtValue).Methods("POST")
	router.HandleFunc("/buy-a-book", incrementBooksBought).Methods("GET")
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), router))
}
