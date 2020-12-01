package cds

import (
	"fmt"
	_ "strings"
	"time"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	_ "github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// clusterConnectTimeout is the timeout duration used by Envoy to timeout connections to the cluster
	clusterConnectTimeout = 1 * time.Second
)

// getRemoteServiceCluster returns an Envoy Cluster corresponding to the remote service
func getRemoteServiceCluster(remoteService, localService service.MeshServicePort, cfg configurator.Configurator) (*xds_cluster.Cluster, error) {
	clusterName := remoteService.String()
	/* WITESAND_TLS_DISABLE
	sni := remoteService.GetMeshService().GetCommonName().String()
	marshalledUpstreamTLSContext, err := envoy.MessageToAny(
		envoy.GetUpstreamTLSContext(localService.GetMeshService(), sni))
	if err != nil {
		return nil, err
	}
	*/

	remoteCluster := &xds_cluster.Cluster{
		Name:           clusterName,
		ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
		/* WITESAND_TLS_DISABLE
		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledUpstreamTLSContext,
			},
		},
		*/
		ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
		Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
	}

	if cfg.IsPermissiveTrafficPolicyMode() {
		// Since no traffic policies exist with permissive mode, rely on cluster provided service discovery.
		remoteCluster.ClusterDiscoveryType = &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_ORIGINAL_DST}
		remoteCluster.LbPolicy = xds_cluster.Cluster_CLUSTER_PROVIDED
	} else {
		// Configure service discovery based on traffic policies
		remoteCluster.ClusterDiscoveryType = &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS}
		remoteCluster.EdsClusterConfig = &xds_cluster.Cluster_EdsClusterConfig{EdsConfig: envoy.GetADSConfigSource()}
		remoteCluster.LbPolicy = xds_cluster.Cluster_ROUND_ROBIN
	}

	return remoteCluster, nil
}

// getOutboundPassthroughCluster returns an Envoy cluster that is used for outbound passthrough traffic
func getOutboundPassthroughCluster() *xds_cluster.Cluster {
	return &xds_cluster.Cluster{
		Name:           envoy.OutboundPassthroughCluster,
		ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_ORIGINAL_DST,
		},
		LbPolicy:             xds_cluster.Cluster_CLUSTER_PROVIDED,
		ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
		Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
	}
}

// getLocalServiceCluster returns an Envoy Cluster corresponding to the local service
func getLocalServiceCluster(catalog catalog.MeshCataloger, proxyServiceName service.MeshService) ([]*xds_cluster.Cluster, error) {
	xdsClusters := make([]*xds_cluster.Cluster, 0)
	endpoints, err := catalog.ListEndpointsForService(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get endpoints for service %s", proxyServiceName)
		return nil, err
	}

	for _, ep := range endpoints {
		// final newClusterName shuould be something like "bookstore/bookstore-v1/80-local"
		newClusterName := fmt.Sprintf("%s/%d%s", proxyServiceName, ep.Port, envoy.LocalClusterSuffix)
		log.Debug().Msgf("clusterName:%s newClusterName", proxyServiceName, newClusterName)
		xdsCluster := &xds_cluster.Cluster{
			// The name must match the domain being cURLed in the demo
			Name:           newClusterName,
			AltStatName:    newClusterName,
			ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
			LbPolicy:       xds_cluster.Cluster_ROUND_ROBIN,
			RespectDnsTtl:  true,
			ClusterDiscoveryType: &xds_cluster.Cluster_Type{
				Type: xds_cluster.Cluster_STRICT_DNS,
			},
			DnsLookupFamily: xds_cluster.Cluster_V4_ONLY,
			LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
				// NOTE: results.MeshService is the top level service that is cURLed.
				ClusterName: newClusterName,
				Endpoints:   []*xds_endpoint.LocalityLbEndpoints{
					// Filled based on discovered endpoints for the service
				},
			},
			ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
			Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
		}

		localityEndpoint := &xds_endpoint.LocalityLbEndpoints{
			Locality: &xds_core.Locality{
				Zone: "zone",
			},
			LbEndpoints: []*xds_endpoint.LbEndpoint{{
				HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
					Endpoint: &xds_endpoint.Endpoint{
						Address: envoy.GetAddress(constants.WildcardIPAddr, uint32(ep.Port)),
					},
				},
				LoadBalancingWeight: &wrappers.UInt32Value{
					Value: constants.ClusterWeightAcceptAll, // Local cluster accepts all traffic
				},
			}},
		}
		xdsCluster.LoadAssignment.Endpoints = append(xdsCluster.LoadAssignment.Endpoints, localityEndpoint)
		xdsClusters = append(xdsClusters, xdsCluster)
	}

	log.Debug().Msgf("count %d xdsClusters:%+v", len(xdsClusters), xdsClusters)

	return xdsClusters, nil
}

// getPrometheusCluster returns an Envoy Cluster responsible for scraping metrics by Prometheus
func getPrometheusCluster() xds_cluster.Cluster {
	return xds_cluster.Cluster{
		// The name must match the domain being cURLed in the demo
		Name:           constants.EnvoyMetricsCluster,
		AltStatName:    constants.EnvoyMetricsCluster,
		ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_STATIC,
		},
		LbPolicy: xds_cluster.Cluster_ROUND_ROBIN,
		LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
			// NOTE: results.MeshService is the top level service that is cURLed.
			ClusterName: constants.EnvoyMetricsCluster,
			Endpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
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
		ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
		Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
	}
}
