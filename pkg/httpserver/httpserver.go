package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"

	"github.com/open-service-mesh/osm/pkg/health"
	"github.com/open-service-mesh/osm/pkg/metricsstore"
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
		glog.Infof("Starting API Server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil {
			glog.Fatal("Failed to start API server", err)
		}
	}()
}

func (s *httpServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		glog.Error("Unable to shutdown API server gracefully", err)
	}
}
