package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/openservicemesh/osm/demo/cmd/common"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log        = logger.NewPretty("bookwarehouse")
	identity   = flag.String("ident", "unidentified", "the identity of the container where this demo app is running (VM, K8s, etc)")
	port       = flag.Int("port", 80, "port on which this app is listening for incoming HTTP")
	totalBooks = 0
)

func getIdentity() string {
	ident := os.Getenv("IDENTITY")
	if ident == "" {
		if identity != nil {
			ident = *identity
		}
	}
	return ident
}

// restockBooks decreases the balance of the given bookwarehouse account.
func restockBooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.IdentityHeader, getIdentity())

	var numberOfBooks int
	err := json.NewDecoder(r.Body).Decode(&numberOfBooks)
	if err != nil {
		log.Error().Err(err).Msg("Could not decode request body")
		numberOfBooks = 0
	}
	_, _ = w.Write([]byte(fmt.Sprintf("{\"restocked\":%d}", totalBooks)))
	totalBooks = totalBooks + numberOfBooks

	log.Info().Msgf("Restocking bookstore with %d new books; Total so far: %d", numberOfBooks, totalBooks)
	if totalBooks >= 3 {
		fmt.Println(common.Success)
	}
}

func main() {
	flag.Parse()

	//initializing router
	router := mux.NewRouter()

	router.HandleFunc(fmt.Sprintf("/%s", common.RestockWarehouseURL), restockBooks).Methods("POST")
	router.HandleFunc("/", restockBooks).Methods("POST")
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	log.Info().Msgf("Starting BookWarehouse HTTP server on port %d", *port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), router)
	log.Fatal().Err(err).Msgf("Failed to start BookWarehouse HTTP server on port %d", *port)
}
