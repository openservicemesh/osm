package main

import (
	_ "embed"
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
	//go:embed bookthief.html.template
	htmlTpl string
	tmpl    *template.Template
	books   common.BookThiefThievery
	log     = logger.NewPretty(participantName)
	port    = flag.Int("port", 14001, "port on which this app is listening for incoming HTTP")
)

func renderTemplate(w http.ResponseWriter) {
	err := tmpl.Execute(w, map[string]string{
		"Identity":      getIdentity(),
		"BooksStolenV1": fmt.Sprintf("%d", books.BooksStolenV1),
		"BooksStolenV2": fmt.Sprintf("%d", books.BooksStolenV2),
		"BooksStolen":   fmt.Sprintf("%d", books.BooksStolen),
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
	fmt.Printf("%s;  URL: %q;  Count: %d\n", getIdentity(), html.EscapeString(r.URL.Path), books.BooksStolen)
}

func getHandlers() []handler {
	return []handler{
		{"/", getIndex, "GET"},
		{"/raw", common.GetRawGenerator(&books), "GET"},
		{"/reset", reset, "GET"},
	}
}

func reset(w http.ResponseWriter, _ *http.Request) {
	books.BooksStolen = 0
	books.BooksStolenV1 = 0
	books.BooksStolenV2 = 0
	renderTemplate(w)
}
func debugServer() {
	flag.Parse()

	var err error
	tmpl, err = template.New("").Parse(htmlTpl)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse HTML template file")
	}

	router := mux.NewRouter()
	for _, h := range getHandlers() {
		router.HandleFunc(h.path, h.fn).Methods(h.method)
	}
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	log.Info().Msgf("Bookthief running on port %d", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), router)
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
	common.GetBooks(
		participantName,
		meshExpectedResponseCode,
		&books.BooksStolen,
		&books.BooksStolenV1,
		&books.BooksStolenV2,
	)
}
