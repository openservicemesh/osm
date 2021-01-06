package main

import (
	"flag"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/openservicemesh/osm/demo/cmd/common"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/utils"
)

const (
	participantName    = "bookthief"
	httpStatusNotFound = "404"
)

var (
	booksStolen   int64
	booksStolenV1 int64
	booksStolenV2 int64
	log           = logger.NewPretty(participantName)
	port          = flag.Int("port", 80, "port on which this app is listening for incoming HTTP")
	path          = flag.String("path", ".", "path to the HTML template")
)

func renderTemplate(w http.ResponseWriter) {
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/bookthief.html.template", *path))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse HTML template file")
	}
	err = tmpl.Execute(w, map[string]string{
		"Identity":      getIdentity(),
		"BooksStolenV1": fmt.Sprintf("%d", booksStolenV1),
		"BooksStolenV2": fmt.Sprintf("%d", booksStolenV2),
		"BooksStolen":   fmt.Sprintf("%d", booksStolen),
		"Time":          time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Could not render template")
	}
}
func getIdentity() string {
	return utils.GetEnv("IDENTITY", "Bookthief")
}

type handler struct {
	path   string
	fn     func(http.ResponseWriter, *http.Request)
	method string
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w)
	fmt.Printf("%s;  URL: %q;  Count: %d\n", getIdentity(), html.EscapeString(r.URL.Path), booksStolen)
}

func getHandlers() []handler {
	return []handler{
		{"/", getIndex, "GET"},
		{"/reset", reset, "GET"},
	}
}

func reset(w http.ResponseWriter, _ *http.Request) {
	booksStolen = 0
	booksStolenV1 = 0
	booksStolenV2 = 0
	renderTemplate(w)
}
func debugServer() {
	flag.Parse()

	router := mux.NewRouter()
	for _, h := range getHandlers() {
		router.HandleFunc(h.path, h.fn).Methods(h.method)
	}
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	log.Info().Msgf("Bookthief running on port %d", *port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), router)
	log.Fatal().Err(err).Msgf("Failed to start HTTP server on port %d", *port)
}

func main() {
	go debugServer()

	// The bookthief is not allowed to purchase books from the bookstore.
	//
	// Depending on  whether SMI policies or permissive traffic policy is enabled,
	// the HTTP response status code will differ for in-mesh requests.
	//
	// Expected response code when bookthief tries to buy books from the bookstore:
	// 1. With SMI policies: 0
	// 2. With permissive traffic policy: 200
	//
	// When it tries to make an egress request, we expect a 200 response with egress enabled and a 404 response with egress disabled.
	meshExpectedResponseCode := common.GetExpectedResponseCodeFromEnvVar(common.BookthiefExpectedResponseCodeEnvVar, httpStatusNotFound)
	common.GetBooks(participantName, meshExpectedResponseCode, &booksStolen, &booksStolenV1, &booksStolenV2)
}
