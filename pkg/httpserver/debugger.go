package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/debugger"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

// DebugServer is the object wrapper for OSM's HTTP server class
type DebugServer struct {
	Server *http.Server
}

// DebugServerInterface is the interface of the Debug HTTP server.
type DebugServerInterface interface {
	Stop() error
	Start()
}

// RegisterDebugServer registers a go routine to listen to configuration and configure debug server as needed
func RegisterDebugServer(dbgServerInterface DebugServerInterface, cfg configurator.Configurator) {
	// Subscribe to configuration updates
	ch := events.GetPubSubInstance().Subscribe(
		announcements.ConfigMapAdded,
		announcements.ConfigMapDeleted,
		announcements.ConfigMapUpdated)

	// Run config listener
	go func(cfgSubChannel chan interface{}, cf configurator.Configurator, dbgIf DebugServerInterface) {
		// Bootstrap after subscribing
		dbgSrvRunning := false
		if cfg.IsDebugServerEnabled() {
			dbgIf.Start()
			dbgSrvRunning = true
		}

		for {
			<-cfgSubChannel
			isDbgSrvEnabled := cfg.IsDebugServerEnabled()

			if isDbgSrvEnabled && !dbgSrvRunning {
				log.Debug().Msgf("Starting DBG server")
				dbgIf.Start()
				dbgSrvRunning = true
			} else if !isDbgSrvEnabled && dbgSrvRunning {
				log.Debug().Msgf("Stopping DBG server")
				err := dbgIf.Stop()
				if err != nil {
					log.Error().Msgf("Error stopping debug server: %v", err)
					continue
				}
				dbgSrvRunning = false
			}
		}
	}(ch, cfg, dbgServerInterface)
}

// NewDebugHTTPServer creates a new API Server for Debug
func NewDebugHTTPServer(debugServer debugger.DebugConfig, apiPort int32) DebugServerInterface {
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

// Stop halts the DebugServer http.server
func (d *DebugServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeoutDuration)
	defer cancel()
	if err := d.Server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Unable to shutdown Debug server gracefully")
		return err
	}
	return nil
}
