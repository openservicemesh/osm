package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"sync"

	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/tests/e2e"
)

var log = logger.NewPretty("retry")

var mu sync.Mutex
var httpRequests uint32

const retryOn = 555

func retryHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	// Count number of http requests received
	httpRequests++
	mu.Unlock()

	// Return status code that causes retry policy
	if httpRequests <= e2e.NumRetries {
		w.WriteHeader(retryOn)
	}
	_, err := w.Write([]byte(fmt.Sprintf("RequestCount:%v\n", httpRequests)))
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't write number of requests recevied")
	}
}

func main() {
	http.HandleFunc("/", retryHandler)
	err := http.ListenAndServe(":9091", nil)
	log.Fatal().Err(err).Msgf("Failed to start HTTP server on port 9091")
}
