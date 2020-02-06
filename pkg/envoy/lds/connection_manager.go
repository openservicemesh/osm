package lds

import (
	"github.com/deislabs/smc/pkg/envoy"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	wellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/glog"
)

const (
	statPrefix     = "http"
	rdsClusterName = "rds"
)

func getServerConnManager() *envoy_hcm.HttpConnectionManager {
	return &envoy_hcm.HttpConnectionManager{
		StatPrefix: "http",
		CodecType:  envoy_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*envoy_hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
		RouteSpecifier: &envoy_hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &xds.RouteConfiguration{
				VirtualHosts: []*route.VirtualHost{{
					Name:    "backend",
					Domains: []string{"*"},
					Routes: []*route.Route{{
						Match: &route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{
								Prefix: "/",
							},
						},
						Action: &route.Route_Route{
							Route: &route.RouteAction{
								ClusterSpecifier: &route.RouteAction_Cluster{
									Cluster: "bookstore-local",
								},
								// PrefixRewrite: "/stats/prometheus", // well-known Admin API endpoint
							},
						},
					}},
				}},
			},
		},
		AccessLog: envoy.GetAccessLog(),
	}
}

func getConnManagerOutbound() *envoy_hcm.HttpConnectionManager {
	return &envoy_hcm.HttpConnectionManager{
		StatPrefix: statPrefix,
		CodecType:  envoy_hcm.HttpConnectionManager_AUTO,
		HttpFilters: []*envoy_hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
		RouteSpecifier: &envoy_hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &xds.RouteConfiguration{
				VirtualHosts: []*route.VirtualHost{{
					Name:    "envoy_admin",
					Domains: []string{"*"},
					Routes: []*route.Route{{
						Match: &route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{
								Prefix: "/",
							},
						},
						Action: &route.Route_Route{
							Route: &route.RouteAction{
								ClusterSpecifier: &route.RouteAction_Cluster{
									Cluster: "bookstore.mesh",
								},
								// PrefixRewrite: "/stats/prometheus", // well-known Admin API endpoint
							},
						},
					}},
				}},
			},
		},

		/*
				HttpFilters: []*connMgr.HttpFilter{
				{
					Name: wellknown.Router,
				},
			},
		*/
		AccessLog: envoy.GetAccessLog(),
	}
}

func getRouteConfig() *xds.RouteConfiguration {
	routeConfiguration := xds.RouteConfiguration{
		Name: "bookstore_route",
		VirtualHosts: []*route.VirtualHost{{
			Name:    "bookstore_host",
			Domains: []string{"*"},
			Routes:  []*route.Route{},
		}},
	}

	rt := route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
			// Grpc: &route.RouteMatch_GrpcRouteMatchOptions{},
		},

		Action: &route.Route_Route{
			Route: &route.RouteAction{
				// ClusterNotFoundResponseCode: route.RouteAction_SERVICE_UNAVAILABLE,
				/*
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: "bookstore.mesh",
					},
				*/
				ClusterSpecifier: envoy.GetWeightedCluster("bookstore.mesh", 100),
			},
		},
	}
	routeConfiguration.VirtualHosts[0].Routes = append(routeConfiguration.VirtualHosts[0].Routes, &rt)
	glog.Infof("[RDS] Constructed RouteConfiguration: %+v", routeConfiguration)
	return &routeConfiguration
}
