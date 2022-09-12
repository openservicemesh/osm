// Package health implements functionality for readiness and liveness health probes. The probes are served
// by an HTTP server that exposes HTTP paths to probe on, with this package providing the necessary HTTP
// handlers to respond to probe requests.
package health

import (
	"net/http"

	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("health")

// SimpleHandler returns a simple http handler for health checks.
func SimpleHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Health OK")); err != nil {
		log.Error().Err(err).Msg("Error writing bytes for crd-conversion webhook health check handler")
	}
}
