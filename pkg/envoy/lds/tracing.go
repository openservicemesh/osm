package lds

import (
	xds_tracing "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
)

// GetTracingConfig returns a configuration tracing struct for a connection manager to use
func GetTracingConfig(cfg configurator.Configurator) (*xds_hcm.HttpConnectionManager_Tracing, error) {
	zipkinTracingConf := &xds_tracing.ZipkinConfig{
		CollectorCluster:         constants.EnvoyTracingCluster,
		CollectorEndpoint:        cfg.GetTracingEndpoint(),
		CollectorEndpointVersion: xds_tracing.ZipkinConfig_HTTP_JSON,
	}

	zipkinConfMarshalled, err := ptypes.MarshalAny(zipkinTracingConf)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling Zipkin config")
		return nil, err
	}

	tracing := &xds_hcm.HttpConnectionManager_Tracing{
		Verbose: true,
		Provider: &xds_tracing.Tracing_Http{
			// Name must refer to an instantiatable tracing driver
			Name: "envoy.tracers.zipkin",
			ConfigType: &xds_tracing.Tracing_Http_TypedConfig{
				TypedConfig: zipkinConfMarshalled,
			},
		},
	}

	return tracing, nil
}
