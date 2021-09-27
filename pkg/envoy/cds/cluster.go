package cds

import (
	"strings"
	"time"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// replacer used to configure an Envoy cluster's altStatName
var replacer = strings.NewReplacer(".", "_", ":", "_")

// getUpstreamServiceCluster returns an Envoy Cluster corresponding to the given upstream service
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func getUpstreamServiceCluster(downstreamIdentity identity.ServiceIdentity, config trafficpolicy.MeshClusterConfig) *xds_cluster.Cluster {
	HTTP2ProtocolOptions, err := envoy.GetHTTP2ProtocolOptions()
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HTTP2ProtocolOptions for upstream cluster %s", config.Name)
		return nil
	}

	marshalledUpstreamTLSContext, err := ptypes.MarshalAny(
		envoy.GetUpstreamTLSContext(downstreamIdentity, config.Service))
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling UpstreamTLSContext for upstream cluster %s", config.Name)
		return nil
	}

	remoteCluster := &xds_cluster.Cluster{
		Name:                          config.Name,
		TypedExtensionProtocolOptions: HTTP2ProtocolOptions,
		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledUpstreamTLSContext,
			},
		},
	}

	// Configure service discovery based on traffic policies
	remoteCluster.ClusterDiscoveryType = &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS}
	remoteCluster.EdsClusterConfig = &xds_cluster.Cluster_EdsClusterConfig{EdsConfig: envoy.GetADSConfigSource()}
	remoteCluster.LbPolicy = xds_cluster.Cluster_ROUND_ROBIN

	if config.EnableEnvoyActiveHealthChecks {
		enableHealthChecksOnCluster(remoteCluster, config.Service)
	}
	return remoteCluster
}

// getMulticlusterGatewayUpstreamServiceCluster returns an Envoy Cluster corresponding to the given upstream service for the multicluster gateway
func getMulticlusterGatewayUpstreamServiceCluster(catalog catalog.MeshCataloger, upstreamSvc service.MeshService, withActiveHealthChecks bool) (*xds_cluster.Cluster, error) {
	HTTP2ProtocolOptions, err := envoy.GetHTTP2ProtocolOptions()
	if err != nil {
		return nil, err
	}

	remoteCluster := &xds_cluster.Cluster{
		Name: upstreamSvc.ServerName(),
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_STRICT_DNS,
		},
		LbPolicy:                      xds_cluster.Cluster_ROUND_ROBIN,
		TypedExtensionProtocolOptions: HTTP2ProtocolOptions,
		LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
			ClusterName: upstreamSvc.ServerName(),
			Endpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: envoy.GetAddress(upstreamSvc.ServerName(), uint32(upstreamSvc.TargetPort)),
							},
						},
					}},
				},
			},
		},
	}

	if withActiveHealthChecks {
		enableHealthChecksOnCluster(remoteCluster, upstreamSvc)
	}
	return remoteCluster, nil
}

func enableHealthChecksOnCluster(cluster *xds_cluster.Cluster, upstreamSvc service.MeshService) {
	cluster.HealthChecks = []*xds_core.HealthCheck{
		{
			Timeout:            durationpb.New(1 * time.Second),
			Interval:           durationpb.New(10 * time.Second),
			HealthyThreshold:   wrapperspb.UInt32(1),
			UnhealthyThreshold: wrapperspb.UInt32(3),
			HealthChecker: &xds_core.HealthCheck_HttpHealthCheck_{
				HttpHealthCheck: &xds_core.HealthCheck_HttpHealthCheck{
					Host: upstreamSvc.ServerName(),
					Path: envoy.EnvoyActiveHealthCheckPath,
					RequestHeadersToAdd: []*xds_core.HeaderValueOption{
						{
							Header: &xds_core.HeaderValue{
								Key:   envoy.EnvoyActiveHealthCheckHeaderKey,
								Value: "1",
							},
						},
					},
				},
			},
		},
	}
}

// getLocalServiceCluster returns an Envoy Cluster corresponding to the local service
func getLocalServiceCluster(config trafficpolicy.MeshClusterConfig) *xds_cluster.Cluster {
	HTTP2ProtocolOptions, err := envoy.GetHTTP2ProtocolOptions()
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HTTP2ProtocolOptions for local cluster %s", config.Name)
		return nil
	}

	return &xds_cluster.Cluster{
		// The name must match the domain being cURLed in the demo
		Name:          config.Name,
		AltStatName:   config.Name,
		LbPolicy:      xds_cluster.Cluster_ROUND_ROBIN,
		RespectDnsTtl: true,
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_STRICT_DNS,
		},
		DnsLookupFamily: xds_cluster.Cluster_V4_ONLY,
		LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
			// NOTE: results.MeshService is the top level service that is cURLed.
			ClusterName: config.Name,
			Endpoints: []*xds_endpoint.LocalityLbEndpoints{
				// Filled based on discovered endpoints for the service
				{
					Locality: &xds_core.Locality{
						Zone: "zone",
					},
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: envoy.GetAddress(config.Address, config.Port),
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: constants.ClusterWeightAcceptAll, // Local cluster accepts all traffic
						},
					}},
				},
			},
		},
		TypedExtensionProtocolOptions: HTTP2ProtocolOptions,
	}
}

// getPrometheusCluster returns an Envoy Cluster responsible for scraping metrics by Prometheus
func getPrometheusCluster() *xds_cluster.Cluster {
	return &xds_cluster.Cluster{
		Name:        constants.EnvoyMetricsCluster,
		AltStatName: constants.EnvoyMetricsCluster,
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_STATIC,
		},
		LbPolicy: xds_cluster.Cluster_ROUND_ROBIN,
		LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
			// NOTE: results.MeshService is the top level service that is accessed.
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
	}
}

// getEgressClusters returns a slice of XDS cluster objects for the given egress cluster configs.
// If the cluster config is invalid, an error is logged and the corresponding cluster config is ignored.
func getEgressClusters(clusterConfigs []*trafficpolicy.EgressClusterConfig) []*xds_cluster.Cluster {
	if clusterConfigs == nil {
		return nil
	}

	var egressClusters []*xds_cluster.Cluster
	for _, config := range clusterConfigs {
		switch config.Host {
		case "":
			// Cluster config does not have a Host specified, route it to its original destination.
			// Used for TCP based clusters
			if originalDestinationEgressCluster, err := getOriginalDestinationEgressCluster(config.Name); err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingOrgDstEgressCluster)).
					Msg("Error building the original destination cluster for the given egress cluster config")
			} else {
				egressClusters = append(egressClusters, originalDestinationEgressCluster)
			}
		default:
			// Cluster config has a Host specified, route it based on the Host resolved using DNS.
			// Used for HTTP based clusters
			if cluster, err := getDNSResolvableEgressCluster(config); err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingDNSEgressCluster)).
					Msg("Error building cluster for the given egress cluster config")
			} else {
				egressClusters = append(egressClusters, cluster)
			}
		}
	}

	return egressClusters
}

// getDNSResolvableEgressCluster returns an XDS cluster object that is resolved using DNS for the given egress cluster config.
// If the egress cluster config is invalid, an error is returned.
func getDNSResolvableEgressCluster(config *trafficpolicy.EgressClusterConfig) (*xds_cluster.Cluster, error) {
	if config == nil {
		return nil, errors.New("Invalid egress cluster config: nil type")
	}
	if config.Name == "" {
		return nil, errors.New("Invalid egress cluster config: Name unspecified")
	}
	if config.Host == "" {
		return nil, errors.New("Invalid egress cluster config: Host unspecified")
	}
	if config.Port == 0 {
		return nil, errors.New("Invalid egress cluster config: Port unspecified")
	}

	return &xds_cluster.Cluster{
		Name:        config.Name,
		AltStatName: formatAltStatNameForPrometheus(config.Name),
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_STRICT_DNS,
		},
		LbPolicy: xds_cluster.Cluster_ROUND_ROBIN,
		LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
			ClusterName: config.Name,
			Endpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: envoy.GetAddress(config.Host, uint32(config.Port)),
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: constants.ClusterWeightAcceptAll,
						},
					}},
				},
			},
		},
	}, nil
}

// getOriginalDestinationEgressCluster returns an Envoy cluster that routes traffic to its original destination.
// The original destination is the original IP address and port prior to being redirected to the sidecar proxy.
func getOriginalDestinationEgressCluster(name string) (*xds_cluster.Cluster, error) {
	HTTP2ProtocolOptions, err := envoy.GetHTTP2ProtocolOptions()
	if err != nil {
		return nil, err
	}

	return &xds_cluster.Cluster{
		Name: name,
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_ORIGINAL_DST,
		},
		LbPolicy:                      xds_cluster.Cluster_CLUSTER_PROVIDED,
		TypedExtensionProtocolOptions: HTTP2ProtocolOptions,
	}, nil
}

// formatAltStatNameForPrometheus returns an altStatName for a Envoy cluster. If the cluster name contains
// periods or colons the characters must be removed so that the name is correctly interpreted by Envoy when
// generating stats/prometheus. The Envoy cluster's name can remain the same, and the formatted cluster name
// can be assigned to the cluster's altStatName.
func formatAltStatNameForPrometheus(clusterName string) string {
	return replacer.Replace(clusterName)
}

func upstreamClustersFromClusterConfigs(downstreamIdentity identity.ServiceIdentity, configs []*trafficpolicy.MeshClusterConfig) []*xds_cluster.Cluster {
	var clusters []*xds_cluster.Cluster

	for _, c := range configs {
		clusters = append(clusters, getUpstreamServiceCluster(downstreamIdentity, *c))
	}
	return clusters
}

func localClustersFromClusterConfigs(configs []*trafficpolicy.MeshClusterConfig) []*xds_cluster.Cluster {
	var clusters []*xds_cluster.Cluster

	for _, c := range configs {
		clusters = append(clusters, getLocalServiceCluster(*c))
	}
	return clusters
}
