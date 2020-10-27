package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openservicemesh/osm/pkg/debugger"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

// DebugServer is the object wrapper for OSM's HTTP server class
type DebugServer struct {
	Server *http.Server
}

//DebugServerInterface contains debug server functions
type DebugServerInterface interface {
	Stop() error
	Start()
}

// NewDebugHTTPServer creates a new API Server for Debug
func NewDebugHTTPServer(debugServer debugger.DebugServer, apiPort int32) DebugServerInterface {
	return &DebugServer{
		Server: &http.Server{
			Addr:    fmt.Sprintf(":%d", apiPort),
			Handler: NewHealthMux(debugServer.GetHandlers()),
		},
	}
}

// Start runs the Serve operations for the DebugServer http.server on a separate go routine context
func (d *DebugServer) Start() {
	go func() {
		log.Info().Msgf("Starting Debug Server on %s", d.Server.Addr)
		if err := d.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError,
				"Error starting Debug server")
		}
	}()
}

//Stop halts the DebugServer http.server
func (d *DebugServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeoutDuration)
	defer cancel()
	if err := d.Server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Unable to shutdown Debug server gracefully")
		return err
	}
	return nil
}
