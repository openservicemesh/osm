package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/openservicemesh/osm/pkg/kubernetes/events"
	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	contextTimeoutDuration = 5 * time.Second
)

var (
	log = logger.New("http-server")
)

// HTTPServer is the object wrapper for OSM's HTTP server class
type HTTPServer struct {
	started      bool
	server       *http.Server
	httpServeMux *http.ServeMux // Used to restart the server once stopped
	port         uint16         // Used to restart the server once stopped
	stopSyncChan chan struct{}
}

// NewHTTPServer creates a new API server
func NewHTTPServer(port uint16) *HTTPServer {
	serverMux := http.NewServeMux()

	return &HTTPServer{
		started: false,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: serverMux,
		},
		httpServeMux: serverMux,
		port:         port,
		stopSyncChan: make(chan struct{}),
	}
}

// AddHandler adds an HTTP handlers for the given path on the HTTPServer
// For changes to be effective, server requires restart
func (s *HTTPServer) AddHandler(url string, handler http.Handler) {
	s.httpServeMux.Handle(url, handler)
}

// AddHandlers convenience, multi-value AddHandler
func (s *HTTPServer) AddHandlers(handlers map[string]http.Handler) {
	for url, handler := range handlers {
		s.AddHandler(url, handler)
	}
}

// Start starts ListenAndServe on the http server.
// If already started, does nothing
func (s *HTTPServer) Start() error {
	if s.started {
		return nil
	}

	go func() {
		log.Info().Msgf("Starting API Server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError,
				"Error starting HTTP server")
		}
		s.stopSyncChan <- struct{}{}
	}()

	s.started = true
	return nil
}

// Stop halts and resets the http server
// If server is already stopped, does nothing
func (s *HTTPServer) Stop() error {
	if !s.started {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeoutDuration)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Unable to shutdown API server gracefully")
		return err
	}

	// Since we want to free the server, if shutdown succeeded wait for ListenAndServe to return
	<-s.stopSyncChan

	// Free and reset the server, so it can be started again
	s.started = false
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.httpServeMux,
	}

	return nil
}
