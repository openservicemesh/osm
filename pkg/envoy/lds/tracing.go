package lds

import (
	xds_tracing "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
)

// getHTTPTracingConfig returns an HTTP configuration tracing config for the HTTP connection manager to use
func getHTTPTracingConfig(apiEndpoint string) (*xds_hcm.HttpConnectionManager_Tracing, error) {
	zipkinTracingConf := &xds_tracing.ZipkinConfig{
		CollectorCluster:         constants.EnvoyTracingCluster,
		CollectorEndpoint:        apiEndpoint,
		CollectorEndpointVersion: xds_tracing.ZipkinConfig_HTTP_JSON,
	}

	zipkinConfMarshalled, err := anypb.New(zipkinTracingConf)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling Zipkin config")
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
