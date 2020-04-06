package cds

import (
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoyEndpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

const (
	weightAcceptAll   uint32        = 100
	connectionTimeout time.Duration = 1 * time.Second
)

func getServiceClusterLocal(catalog catalog.MeshCataloger, proxyService endpoint.NamespacedService, clusterName string) xds.Cluster {
	xdsCluster := xds.Cluster{
		// The name must match the domain being cURLed in the demo
		Name:                          clusterName,
		AltStatName:                   clusterName,
		ConnectTimeout:                ptypes.DurationProto(connectionTimeout),
		LbPolicy:                      xds.Cluster_ROUND_ROBIN,
		RespectDnsTtl:                 true,
		DrainConnectionsOnHostRemoval: true,
		ClusterDiscoveryType: &xds.Cluster_Type{
			Type: xds.Cluster_STRICT_DNS,
		},
		DnsLookupFamily: xds.Cluster_V4_ONLY,
		LoadAssignment: &xds.ClusterLoadAssignment{
			// NOTE: results.ServiceName is the top level service that is cURLed.
			ClusterName: clusterName,
			Endpoints:   []*envoyEndpoint.LocalityLbEndpoints{
				// Filled based on discovered endpoints for the service
			},
		},
	}

	svcEndpoints, _ := catalog.ListEndpoints(proxyService)
	for _, svcEp := range svcEndpoints {
		for _, ep := range svcEp.Endpoints {
			localityEndpoint := &envoyEndpoint.LocalityLbEndpoints{
				Locality: &core.Locality{
					Zone: "zone",
				},
				LbEndpoints: []*envoyEndpoint.LbEndpoint{{
					HostIdentifier: &envoyEndpoint.LbEndpoint_Endpoint{
						Endpoint: &envoyEndpoint.Endpoint{
							Address: envoy.GetAddress(constants.WildcardIPAddr, uint32(ep.Port)),
						},
					},
					LoadBalancingWeight: &wrappers.UInt32Value{
						Value: weightAcceptAll, // Local cluster accepts all traffic
					},
				}},
			}
			xdsCluster.LoadAssignment.Endpoints = append(xdsCluster.LoadAssignment.Endpoints, localityEndpoint)
		}
	}

	return xdsCluster
}

func getPrometheusCluster(clusterName string) xds.Cluster {
	return xds.Cluster{
		// The name must match the domain being cURLed in the demo
		Name:           clusterName,
		AltStatName:    clusterName,
		ConnectTimeout: ptypes.DurationProto(connectionTimeout),
		ClusterDiscoveryType: &xds.Cluster_Type{
			Type: xds.Cluster_STATIC,
		},
		LbPolicy: xds.Cluster_ROUND_ROBIN,
		LoadAssignment: &xds.ClusterLoadAssignment{
			// NOTE: results.ServiceName is the top level service that is cURLed.
			ClusterName: clusterName,
			Endpoints: []*envoyEndpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*envoyEndpoint.LbEndpoint{{
						HostIdentifier: &envoyEndpoint.LbEndpoint_Endpoint{
							Endpoint: &envoyEndpoint.Endpoint{
								Address: envoy.GetAddress(constants.LocalhostIPAddress, constants.EnvoyAdminPort),
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: weightAcceptAll,
						},
					}},
				},
			},
		},
	}
}
