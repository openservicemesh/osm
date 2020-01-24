package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

var identity = flag.String("ident", "unidentified", "the identity of the container where this demo app is running (VM, K8s, etc)")

var port = flag.Int("port", 80, "port on which this app is listening for incoming HTTP")
var path = flag.String("path", ".", "path to the HTML template")
var counter int
var tmpl *template.Template

// getCurrentCounter gets the value of the counter
func getCurrentCounter(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Counter", fmt.Sprintf("%d", counter))
	ident := os.Getenv("IDENTITY")
	if ident == "" {
		if identity != nil {
			ident = *identity
		}
	}
	w.Header().Set("Identity", fmt.Sprintf("%s", ident))
	tmpl.Execute(w, map[string]string{"Identity": ident, "Counter": fmt.Sprintf("%d", counter)})
	fmt.Printf("%s;  URL: %q;  Count: %d\n", ident, html.EscapeString(r.URL.Path), counter)
}

// updateCounterValue updates the counter value to the one specified by the user
func updateCounterValue(w http.ResponseWriter, r *http.Request) {

	var updatedCounter int
	json.NewDecoder(r.Body).Decode(&updatedCounter)
	counter = updatedCounter
	w.Header().Set("Counter", fmt.Sprintf("%d", counter))
	ident := os.Getenv("IDENTITY")
	if ident == "" {
		if identity != nil {
			ident = *identity
		}
	}
	w.Header().Set("Identity", fmt.Sprintf("%s", ident))
	tmpl.Execute(w, map[string]string{"Identity": ident, "Counter": fmt.Sprintf("%d", counter)})
	fmt.Printf("%s;  URL: %q;  Count: %d\n", ident, html.EscapeString(r.URL.Path), counter)
}

// incrementCounter increments the value of the counter
func incrementCounter(w http.ResponseWriter, r *http.Request) {

	counter++
	w.Header().Set("Counter", fmt.Sprintf("%d", counter))
	ident := os.Getenv("IDENTITY")
	if ident == "" {
		if identity != nil {
			ident = *identity
		}
	}
	w.Header().Set("Identity", fmt.Sprintf("%s", ident))
	tmpl.Execute(w, map[string]string{"Identity": ident, "Counter": fmt.Sprintf("%d", counter)})
	fmt.Printf("%s;  URL: %q;  Count: %d\n", ident, html.EscapeString(r.URL.Path), counter)
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
	tmpl, err = template.ParseFiles(fmt.Sprintf("%s/counter.html.template", *path))
	if err != nil {
		log.Fatal(err)
	}
	counter = 1

	//initializing router
	router := mux.NewRouter()

	//endpoints
	router.HandleFunc("/counter", getCurrentCounter).Methods("GET")
	router.HandleFunc("/counter", updateCounterValue).Methods("POST")
	router.HandleFunc("/incrementcounter", incrementCounter).Methods("GET")
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), router))
}
