package cds

import (
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoyEndpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// clusterConnectTimeout is the timeout duration used by Envoy to timeout connections to the cluster
	clusterConnectTimeout = 1 * time.Second
)

// getRemoteServiceCluster returns an Envoy Cluster corresponding to the remote service
func getRemoteServiceCluster(remoteService, localService service.NamespacedService) (*xds.Cluster, error) {
	clusterName := remoteService.String()
	marshalledUpstreamTLSContext, err := envoy.MessageToAny(
		envoy.GetUpstreamTLSContext(localService, remoteService.GetCommonName().String()))
	if err != nil {
		return nil, err
	}
	return &xds.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       ptypes.DurationProto(clusterConnectTimeout),
		LbPolicy:             xds.Cluster_ROUND_ROBIN,
		ClusterDiscoveryType: &xds.Cluster_Type{Type: xds.Cluster_EDS},
		EdsClusterConfig:     &xds.Cluster_EdsClusterConfig{EdsConfig: envoy.GetADSConfigSource()},
		TransportSocket: &core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &core.TransportSocket_TypedConfig{
				TypedConfig: marshalledUpstreamTLSContext,
			},
		},
	}, nil
}

// getOutboundPassthroughCluster returns an Envoy cluster that is used for outbound passthrough traffic
func getOutboundPassthroughCluster() *xds.Cluster {
	return &xds.Cluster{
		Name:           envoy.OutboundPassthroughCluster,
		ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
		ClusterDiscoveryType: &xds.Cluster_Type{
			Type: xds.Cluster_ORIGINAL_DST,
		},
		LbPolicy: xds.Cluster_CLUSTER_PROVIDED,
	}
}

// getLocalServiceCluster returns an Envoy Cluster corresponding to the local service
func getLocalServiceCluster(catalog catalog.MeshCataloger, proxyServiceName service.NamespacedService, clusterName string) (*xds.Cluster, error) {
	xdsCluster := xds.Cluster{
		// The name must match the domain being cURLed in the demo
		Name:                          clusterName,
		AltStatName:                   clusterName,
		ConnectTimeout:                ptypes.DurationProto(clusterConnectTimeout),
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

// getPrometheusCluster returns an Envoy Cluster responsible for scraping metrics by Prometheus
func getPrometheusCluster() xds.Cluster {
	return xds.Cluster{
		// The name must match the domain being cURLed in the demo
		Name:           constants.EnvoyMetricsCluster,
		AltStatName:    constants.EnvoyMetricsCluster,
		ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
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
