package debugger

import (
	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/httpserver"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

// StartDebugServerConfigListener registers a go routine to listen to configuration and configure debug server as needed
func (d *DebugConfig) StartDebugServerConfigListener() {
	// Subscribe to configuration updates
	ch := events.GetPubSubInstance().Subscribe(
		announcements.ConfigMapAdded,
		announcements.ConfigMapDeleted,
		announcements.ConfigMapUpdated)

	// This is the Debug server
	httpDebugServer := httpserver.NewHTTPServer(constants.DebugPort)
	httpDebugServer.AddHandlers(d.GetHandlers())

	// Run config listener
	go func(cfgSubChannel chan interface{}, dConf *DebugConfig, httpServ *httpserver.HTTPServer) {
		// Bootstrap after subscribing
		started := false

		if d.configurator.IsDebugServerEnabled() {
			if err := httpDebugServer.Start(); err != nil {
				log.Error().Err(err).Msgf("error starting debug server")
			}
			started = true
		}

		for {
			<-cfgSubChannel
			isDbgSrvEnabled := d.configurator.IsDebugServerEnabled()

			if isDbgSrvEnabled && !started {
				if err := httpDebugServer.Start(); err != nil {
					log.Error().Err(err).Msgf("error starting debug server")
				} else {
					started = true
				}
			}
			if !isDbgSrvEnabled && started {
				if err := httpDebugServer.Stop(); err != nil {
					log.Error().Err(err).Msgf("error stopping debug server")
				} else {
					started = false
				}
			}
		}
	}(ch, d, httpDebugServer)
}
