package lds

import (
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	wellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/route"
)

const (
	statPrefix = "http"
)

// getRdsHttpClientConnectionFilter gets the required route configuration for the envoy on the source service
func getRdsHTTPClientConnectionFilter() *envoy_hcm.HttpConnectionManager {
	return &envoy_hcm.HttpConnectionManager{
		StatPrefix: statPrefix,
		CodecType:  envoy_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*envoy_hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
		RouteSpecifier: &envoy_hcm.HttpConnectionManager_Rds{
			Rds: &envoy_hcm.Rds{
				ConfigSource:    envoy.GetADSConfigSource(),
				RouteConfigName: route.SourceRouteConfig,
			},
		},
		AccessLog: envoy.GetAccessLog(),
	}
}

// getRdsHttpClientConnectionFilter gets the required route configuration for the envoy on the destination service
func getRdsHTTPServerConnectionFilter() *envoy_hcm.HttpConnectionManager {
	return &envoy_hcm.HttpConnectionManager{
		StatPrefix: statPrefix,
		CodecType:  envoy_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*envoy_hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
		RouteSpecifier: &envoy_hcm.HttpConnectionManager_Rds{
			Rds: &envoy_hcm.Rds{
				ConfigSource:    envoy.GetADSConfigSource(),
				RouteConfigName: route.DestinationRouteConfig,
			},
		},
		AccessLog: envoy.GetAccessLog(),
	}
}
