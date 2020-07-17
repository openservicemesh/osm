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
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/featureflags"
	"github.com/open-service-mesh/osm/pkg/service"
)

const (
	connectionTimeout time.Duration = 1 * time.Second
)

func getServiceClusterLocal(catalog catalog.MeshCataloger, proxyServiceName service.NamespacedService, clusterName string) (*xds.Cluster, error) {
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

	log.Trace().Msgf("Backpressure: %t", featureflags.IsBackpressureEnabled())

	if featureflags.IsBackpressureEnabled() {
		log.Info().Msgf("Enabling backpressure local service cluster")
		// Backpressure CRD only has one backpressure obj as a global config
		// TODO: Add specific backpressure settings for individual clients
		backpressures := catalog.GetSMISpec().ListBackpressures()
		log.Trace().Msgf("Backpressures (%d): %+v", len(backpressures), backpressures)

		if len(backpressures) > 0 {
			log.Trace().Msgf("Backpressure Spec: %+v", backpressures[0].Spec)
			xdsCluster.MaxRequestsPerConnection = &wrappers.UInt32Value{
				Value: backpressures[0].Spec.MaxRequestsPerConnection,
			}
		}
	}

	endpoints, err := catalog.ListEndpointsForService(service.Name(proxyServiceName.String()))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get endpoints for service %s", proxyServiceName)
		return nil, err
	}

	for _, ep := range endpoints {
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
					Value: constants.ClusterWeightAcceptAll, // Local cluster accepts all traffic
				},
			}},
		}
		xdsCluster.LoadAssignment.Endpoints = append(xdsCluster.LoadAssignment.Endpoints, localityEndpoint)
	}

	return &xdsCluster, nil
}

func getPrometheusCluster() xds.Cluster {
	return xds.Cluster{
		// The name must match the domain being cURLed in the demo
		Name:           constants.EnvoyMetricsCluster,
		AltStatName:    constants.EnvoyMetricsCluster,
		ConnectTimeout: ptypes.DurationProto(connectionTimeout),
		ClusterDiscoveryType: &xds.Cluster_Type{
			Type: xds.Cluster_STATIC,
		},
		LbPolicy: xds.Cluster_ROUND_ROBIN,
		LoadAssignment: &xds.ClusterLoadAssignment{
			// NOTE: results.ServiceName is the top level service that is cURLed.
			ClusterName: constants.EnvoyMetricsCluster,
			Endpoints: []*envoyEndpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*envoyEndpoint.LbEndpoint{{
						HostIdentifier: &envoyEndpoint.LbEndpoint_Endpoint{
							Endpoint: &envoyEndpoint.Endpoint{
								Address: envoy.GetAddress(constants.LocalhostIPAddress, constants.EnvoyAdminPort),
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: constants.ClusterWeightAcceptAll,
						},
					}},
				},
			},
		},
	}
}
