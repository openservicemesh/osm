package debugger

import (
	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/httpserver"
	"github.com/openservicemesh/osm/pkg/k8s/events"
)

// StartDebugServerConfigListener registers a go routine to listen to configuration and configure debug server as needed
func (d *DebugConfig) StartDebugServerConfigListener(stop chan struct{}) {
	// This is the Debug server
	httpDebugServer := httpserver.NewHTTPServer(constants.DebugPort)
	httpDebugServer.AddHandlers(d.GetHandlers())

	kubePubSub := d.msgBroker.GetKubeEventPubSub()
	meshCfgUpdateChan := kubePubSub.Sub(announcements.MeshConfigUpdated.String())
	defer d.msgBroker.Unsub(kubePubSub, meshCfgUpdateChan)

	started := false
	if d.configurator.IsDebugServerEnabled() {
		if err := httpDebugServer.Start(); err != nil {
			log.Error().Err(err).Msgf("error starting debug server")
		}
		started = true
	}

	for {
		select {
		case event := <-meshCfgUpdateChan:
			msg, ok := event.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Error casting to PubSubMessage, got type %T", msg)
				continue
			}

			prevSpec := msg.OldObj.(*configv1alpha3.MeshConfig).Spec
			newSpec := msg.NewObj.(*configv1alpha3.MeshConfig).Spec

			if prevSpec.Observability.EnableDebugServer == newSpec.Observability.EnableDebugServer {
				continue
			}

			enableDbgServer := newSpec.Observability.EnableDebugServer
			if enableDbgServer && !started {
				if err := httpDebugServer.Start(); err != nil {
					log.Error().Err(err).Msgf("error starting debug server")
				} else {
					started = true
				}
			} else if !enableDbgServer && started {
				if err := httpDebugServer.Stop(); err != nil {
					log.Error().Err(err).Msgf("error stopping debug server")
				} else {
					started = false
				}
			}

		case <-stop:
			log.Info().Msg("Received stop signal, exiting debug server config listener")
			return
		}
	}
}
