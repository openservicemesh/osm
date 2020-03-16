package lds

import (
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/open-service-mesh/osm/pkg/envoy"
)

const (
	statPrefix = "http"
)

func getHTTPConnectionManager(routeName string) *envoy_hcm.HttpConnectionManager {
	return &envoy_hcm.HttpConnectionManager{
		StatPrefix: statPrefix,
		CodecType:  envoy_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*envoy_hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
		RouteSpecifier: &envoy_hcm.HttpConnectionManager_Rds{
			Rds: &envoy_hcm.Rds{
				ConfigSource:    envoy.GetADSConfigSource(),
				RouteConfigName: routeName,
			},
		},
		AccessLog: envoy.GetAccessLog(),
	}
}
