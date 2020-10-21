package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/openservicemesh/osm/pkg/debugger"
	"github.com/openservicemesh/osm/pkg/health"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/version"
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

// DebugServer is the object wrapper for OSM's HTTP server class
type DebugServer struct {
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
func NewHTTPServer(probes []health.Probes, httpProbes []health.HTTPProbe, metricStore metricsstore.MetricStore, apiPort int32) *HTTPServer {
	handlers := map[string]http.Handler{
		"/health/ready": health.ReadinessHandler(probes, httpProbes),
		"/health/alive": health.LivenessHandler(probes, httpProbes),
		"/metrics":      metricStore.Handler(),
		"/version":      getVersionHandler(),
	}

	return &HTTPServer{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", apiPort),
			Handler: NewHealthMux(handlers),
		},
	}
}

// NewDebugHTTPServer creates a new API Server for Debug
func NewDebugHTTPServer(debugServer debugger.DebugServer, apiPort int32) *DebugServer {
	return &DebugServer{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", apiPort),
			Handler: NewHealthMux(debugServer.GetHandlers()),
		},
	}
}

// Start runs the Serve operations for the http.server on a separate go routine context
func (s *HTTPServer) Start() {
	go func() {
		log.Info().Msgf("Starting API Server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError,
				"Error starting HTTP server")
		}
	}()
}

// Start runs the Serve operations for the DebugServer http.server on a separate go routine context
func (d *DebugServer) Start() {
	go func() {
		log.Info().Msgf("Starting Debug Server on %s", d.server.Addr)
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError,
				"Error starting Debug server")
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

//Stop halts the DebugServer http.server
func (d *DebugServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeoutDuration)
	defer cancel()
	if err := d.server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Unable to shutdown Debug server gracefully")
		return err
	}
	return nil
}

// getVersionHandler returns an HTTP handler that returns the version info
func getVersionHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		versionInfo := version.Info{
			Version:   version.Version,
			BuildDate: version.BuildDate,
			GitCommit: version.GitCommit,
		}

		if jsonVersionInfo, err := json.Marshal(versionInfo); err != nil {
			log.Error().Err(err).Msgf("Error marshaling version info struct: %+v", versionInfo)
		} else {
			_, _ = fmt.Fprint(w, string(jsonVersionInfo))
		}
	})
}
