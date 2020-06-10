package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/open-service-mesh/osm/pkg/debugger"
	"github.com/open-service-mesh/osm/pkg/health"
	"github.com/open-service-mesh/osm/pkg/metricsstore"
)

const (
	contextTimeoutDuration = 5 * time.Second
)

// NewHealthMux makes a new *http.ServeMux
func NewHealthMux(handlers map[string]http.Handler) *http.ServeMux {
	router := http.NewServeMux()
	for url, handler := range handlers {
		router.Handle(url, handler)
	}

	return router
}

// NewHTTPServer creates a new API server
func NewHTTPServer(somethingWithProbes health.Probes, metricStore metricsstore.MetricStore, apiPort int32, debugServer debugger.DebugServer) HTTPServer {
	handlers := map[string]http.Handler{
		"/health/ready": health.ReadinessHandler(somethingWithProbes),
		"/health/alive": health.LivenessHandler(somethingWithProbes),
		"/metrics":      metricStore.Handler(),
	}

	if debugServer != nil {
		for url, handler := range debugServer.GetHandlers() {
			handlers[url] = handler
		}
	}

	return &httpServer{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", apiPort),
			Handler: NewHealthMux(handlers),
		},
	}
}

func (s *httpServer) Start() {
	go func() {
		log.Info().Msgf("Starting API Server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start API server")
		}
	}()
}

func (s *httpServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeoutDuration)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Unable to shutdown API server gracefully")
	}
}
