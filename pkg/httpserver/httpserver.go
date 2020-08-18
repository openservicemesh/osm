package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/openservicemesh/osm/pkg/debugger"
	"github.com/openservicemesh/osm/pkg/health"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

const (
	contextTimeoutDuration = 5 * time.Second
)

var (
	log = logger.New("http-server")
)

// HTTPServer is the object wrapper for OSM's HTTP server class
type HTTPServer struct {
	server *http.Server
}

// NewHealthMux makes a new *http.ServeMux
func NewHealthMux(handlers map[string]http.Handler) *http.ServeMux {
	router := http.NewServeMux()
	for url, handler := range handlers {
		router.Handle(url, handler)
	}

	return router
}

// NewHTTPServer creates a new API server
func NewHTTPServer(probes health.Probes, metricStore metricsstore.MetricStore, apiPort int32, debugServer debugger.DebugServer) *HTTPServer {
	handlers := map[string]http.Handler{
		"/health/ready": health.ReadinessHandler(probes),
		"/health/alive": health.LivenessHandler(probes),
		"/metrics":      metricStore.Handler(),
	}

	if debugServer != nil {
		for url, handler := range debugServer.GetHandlers() {
			handlers[url] = handler
		}
	}

	return &HTTPServer{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", apiPort),
			Handler: NewHealthMux(handlers),
		},
	}
}

// Start runs the Serve operations for the http.server on a separate go routine context
func (s *HTTPServer) Start() {
	go func() {
		log.Info().Msgf("Starting API Server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start API server")
		}
	}()
}

// Stop halts the http.server
func (s *HTTPServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeoutDuration)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Unable to shutdown API server gracefully")
		return err
	}
	return nil
}
