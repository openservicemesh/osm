package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/open-service-mesh/osm/pkg/health"
	"github.com/open-service-mesh/osm/pkg/metricsstore"
)

const (
	contextTimeoutDuration = 5 * time.Second
)

// HTTPServer serving probes and metrics
type HTTPServer interface {
	Start()
	Stop()
}

type httpServer struct {
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

// NewHTTPServer creates a new api server
func NewHTTPServer(somethingWithProbes health.Probes, metricStore metricsstore.MetricStore, apiPort string, debugInfo func() http.Handler) HTTPServer {
	return &httpServer{
		server: &http.Server{
			Addr: fmt.Sprintf(":%s", apiPort),
			Handler: NewHealthMux(map[string]http.Handler{
				"/health/ready": health.ReadinessHandler(somethingWithProbes),
				"/health/alive": health.LivenessHandler(somethingWithProbes),
				"/metrics":      metricStore.Handler(),
				"/debug":        debugInfo(),
			}),
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
