package main

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
)

// StartGlobalLogLevelHandler registers a listener to meshconfig events and log level changes,
// and applies new log level at global scope
func StartGlobalLogLevelHandler(cfg configurator.Configurator, stop <-chan struct{}) {
	meshConfigChannel := events.GetPubSubInstance().Subscribe(
		announcements.MeshConfigAdded,
		announcements.MeshConfigDeleted,
		announcements.MeshConfigUpdated)

	// Run config listener
	// Bootstrap after subscribing
	currentLogLevel := constants.DefaultOSMLogLevel
	logLevel := cfg.GetOSMLogLevel()
	log.Info().Msgf("Setting initial log level from meshconfig: %s", logLevel)
	err := logger.SetLogLevel(logLevel)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrSettingLogLevel.String()).
			Msg("Error setting initial log level from meshconfig")
	} else {
		currentLogLevel = logLevel
	}

	go func() {
		for {
			select {
			case <-meshConfigChannel:
				logLevel := cfg.GetOSMLogLevel()
				if logLevel != currentLogLevel {
					err := logger.SetLogLevel(logLevel)
					if err != nil {
						log.Error().Err(err).Str(errcode.Kind, errcode.ErrSettingLogLevel.String()).
							Msg("Error setting log level from meshconfig")
					} else {
						log.Info().Msgf("Global log level changed to: %s", logLevel)
						currentLogLevel = logLevel
					}
				}
			case <-stop:
				return
			}
		}
	}()
}
