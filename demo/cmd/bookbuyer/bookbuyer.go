package main

import (
	_ "embed"
	"flag"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"

	"github.com/openservicemesh/osm/demo/cmd/common"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/utils"
)

const (
	participantName = "bookbuyer"
)

var (
	//go:embed bookbuyer.html.template
	htmlTpl           string
	tmpl              *template.Template
	books             common.BookBuyerPurchases
	wg                sync.WaitGroup
	log               = logger.NewPretty(participantName)
	port              = flag.Int("port", 14001, "port on which this app is listening for incoming HTTP")
	numConnectionsStr = utils.GetEnv("CI_CLIENT_CONCURRENT_CONNECTIONS", "1")
)

type handler struct {
	path   string
	fn     func(http.ResponseWriter, *http.Request)
	method string
}

func getIdentity() string {
	return utils.GetEnv("IDENTITY", "Bookbuyer")
}

func renderTemplate(w http.ResponseWriter) {
	err := tmpl.Execute(w, map[string]string{
		"Identity":      getIdentity(),
		"BooksBoughtV1": fmt.Sprintf("%d", books.BooksBoughtV1),
		"BooksBoughtV2": fmt.Sprintf("%d", books.BooksBoughtV2),
		"BooksBought":   fmt.Sprintf("%d", books.BooksBought),
		"Time":          time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Could not render template")
	}
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w)
	fmt.Printf("%s;  URL: %q;  Count: %d\n", getIdentity(), html.EscapeString(r.URL.Path), books.BooksBought)
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
	log.Info().Msgf("Web server running on port %d", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), router)
	log.Fatal().Err(err).Msgf("Failed to start HTTP server on port %d", *port)
}

func getHandlers() []handler {
	return []handler{
		{"/", getIndex, "GET"},
		{"/raw", common.GetRawGenerator(&books), "GET"},
		{"/reset", reset, "GET"},
	}
}

func reset(w http.ResponseWriter, _ *http.Request) {
	atomic.StoreInt64(&books.BooksBought, 0)
	atomic.StoreInt64(&books.BooksBoughtV1, 0)
	atomic.StoreInt64(&books.BooksBoughtV2, 0)
	renderTemplate(w)
}

func getBooksWrapper(wg *sync.WaitGroup) {
	defer wg.Done()

	meshExpectedResponseCode := http.StatusOK
	common.GetBooks(
		participantName,
		meshExpectedResponseCode,
		&books.BooksBought,
		&books.BooksBoughtV1,
		&books.BooksBoughtV2,
	)
}

func main() {
	go debugServer()

	numConnections, err := strconv.Atoi(numConnectionsStr)
	if err != nil {
		fmt.Printf("Error: invalid value for number of bookstore connections: %s", numConnectionsStr)
		numConnections = 1
	}

	// This is the bookbuyer.  When it tries to buy books from the bookstore - we expect it to see 200 responses.
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		fmt.Printf("Starting bookbuyer connection #%d", i)
		go getBooksWrapper(&wg)
	}

	wg.Wait()
}
