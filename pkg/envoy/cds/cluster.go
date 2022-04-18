package cds

import (
	"math"
	"strings"
	"time"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	extensions_upstream_http "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

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
func getUpstreamServiceCluster(downstreamIdentity identity.ServiceIdentity, config trafficpolicy.MeshClusterConfig, sidecarSpec configv1alpha3.SidecarSpec) *xds_cluster.Cluster {
	httpProtocolOptions := getDefaultHTTPProtocolOptions()

	marshalledUpstreamTLSContext, err := anypb.New(
		envoy.GetUpstreamTLSContext(downstreamIdentity, config.Service, sidecarSpec))
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling UpstreamTLSContext for upstream cluster %s", config.Name)
		return nil
	}

	upstreamCluster := &xds_cluster.Cluster{
		Name: config.Name,
		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledUpstreamTLSContext,
			},
		},
	}

	// Configure service discovery based on traffic policies
	upstreamCluster.ClusterDiscoveryType = &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS}
	upstreamCluster.EdsClusterConfig = &xds_cluster.Cluster_EdsClusterConfig{EdsConfig: envoy.GetADSConfigSource()}
	upstreamCluster.LbPolicy = xds_cluster.Cluster_ROUND_ROBIN

	if config.EnableEnvoyActiveHealthChecks {
		enableHealthChecksOnCluster(upstreamCluster, config.Service)
	}

	applyUpstreamTrafficSetting(config.UpstreamTrafficSetting, upstreamCluster, httpProtocolOptions)

	typedHTTPProtocolOptions, err := getTypedHTTPProtocolOptions(httpProtocolOptions)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting typed HTTP protocol options for upstream cluster %s", upstreamCluster.Name)
		return nil
	}
	upstreamCluster.TypedExtensionProtocolOptions = typedHTTPProtocolOptions

	return upstreamCluster
}

// getMulticlusterGatewayUpstreamServiceCluster returns an Envoy Cluster corresponding to the given upstream service for the multicluster gateway
func getMulticlusterGatewayUpstreamServiceCluster(catalog catalog.MeshCataloger, upstreamSvc service.MeshService, withActiveHealthChecks bool) (*xds_cluster.Cluster, error) {
	typedHTTPProtocolOptions, err := getTypedHTTPProtocolOptions(getDefaultHTTPProtocolOptions())
	if err != nil {
		return nil, err
	}

	remoteCluster := &xds_cluster.Cluster{
		Name: upstreamSvc.ServerName(),
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_STRICT_DNS,
		},
		LbPolicy:                      xds_cluster.Cluster_ROUND_ROBIN,
		TypedExtensionProtocolOptions: typedHTTPProtocolOptions,
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
	typedHTTPProtocolOptions, err := getTypedHTTPProtocolOptions(getDefaultHTTPProtocolOptions())
	if err != nil {
		log.Error().Err(err).Msgf("Error getting typed HTTP protocol options for local cluster %s", config.Name)
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
		TypedExtensionProtocolOptions: typedHTTPProtocolOptions,
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
			if originalDestinationEgressCluster, err := getOriginalDestinationEgressCluster(config.Name, config.UpstreamTrafficSetting); err != nil {
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

	httpProtocolOptions := getDefaultHTTPProtocolOptions()

	upstreamCluster := &xds_cluster.Cluster{
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
	}

	applyUpstreamTrafficSetting(config.UpstreamTrafficSetting, upstreamCluster, httpProtocolOptions)

	typedHTTPProtocolOptions, err := getTypedHTTPProtocolOptions(httpProtocolOptions)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting typed HTTP protocol options for egress cluster %s", upstreamCluster.Name)
		return nil, err
	}
	upstreamCluster.TypedExtensionProtocolOptions = typedHTTPProtocolOptions

	return upstreamCluster, nil
}

// getOriginalDestinationEgressCluster returns an Envoy cluster that routes traffic to its original destination.
// The original destination is the original IP address and port prior to being redirected to the sidecar proxy.
func getOriginalDestinationEgressCluster(name string, upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting) (*xds_cluster.Cluster, error) {
	httpProtocolOptions := getDefaultHTTPProtocolOptions()

	upstreamCluster := &xds_cluster.Cluster{
		Name: name,
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_ORIGINAL_DST,
		},
		LbPolicy: xds_cluster.Cluster_CLUSTER_PROVIDED,
	}

	applyUpstreamTrafficSetting(upstreamTrafficSetting, upstreamCluster, httpProtocolOptions)

	typedHTTPProtocolOptions, err := getTypedHTTPProtocolOptions(getDefaultHTTPProtocolOptions())
	if err != nil {
		return nil, err
	}
	upstreamCluster.TypedExtensionProtocolOptions = typedHTTPProtocolOptions

	return upstreamCluster, nil
}

// formatAltStatNameForPrometheus returns an altStatName for a Envoy cluster. If the cluster name contains
// periods or colons the characters must be removed so that the name is correctly interpreted by Envoy when
// generating stats/prometheus. The Envoy cluster's name can remain the same, and the formatted cluster name
// can be assigned to the cluster's altStatName.
func formatAltStatNameForPrometheus(clusterName string) string {
	return replacer.Replace(clusterName)
}

func upstreamClustersFromClusterConfigs(downstreamIdentity identity.ServiceIdentity, configs []*trafficpolicy.MeshClusterConfig, sidecarSpec configv1alpha3.SidecarSpec) []*xds_cluster.Cluster {
	var clusters []*xds_cluster.Cluster

	for _, c := range configs {
		clusters = append(clusters, getUpstreamServiceCluster(downstreamIdentity, *c, sidecarSpec))
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

func getDefaultHTTPProtocolOptions() *extensions_upstream_http.HttpProtocolOptions {
	return &extensions_upstream_http.HttpProtocolOptions{
		UpstreamProtocolOptions: &extensions_upstream_http.HttpProtocolOptions_UseDownstreamProtocolConfig{
			UseDownstreamProtocolConfig: &extensions_upstream_http.HttpProtocolOptions_UseDownstreamHttpConfig{
				Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
			},
		},
	}
}

func getTypedHTTPProtocolOptions(httpProtocolOptions *extensions_upstream_http.HttpProtocolOptions) (map[string]*any.Any, error) {
	marshalledHTTPProtocolOptions, err := anypb.New(httpProtocolOptions)
	if err != nil {
		return nil, err
	}

	return map[string]*any.Any{
		"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": marshalledHTTPProtocolOptions,
	}, nil
}

// getDefaultCircuitBreakerThreshold returns the XDS Circuit Breaker thresholds
// at their max value, effectively disabling circuit breaking
func getDefaultCircuitBreakerThreshold() *xds_cluster.CircuitBreakers_Thresholds {
	return &xds_cluster.CircuitBreakers_Thresholds{
		MaxConnections:     &wrapperspb.UInt32Value{Value: math.MaxUint32},
		MaxRequests:        &wrapperspb.UInt32Value{Value: math.MaxUint32},
		MaxPendingRequests: &wrapperspb.UInt32Value{Value: math.MaxUint32},
		MaxRetries:         &wrapperspb.UInt32Value{Value: math.MaxUint32},
		TrackRemaining:     true,
	}
}

// applyUpstreamTrafficSetting updates the given upstream cluster and HTTP protocol options based on the
// upstream traffic setting provided.
// It applies the default circuit breaker thresholds to the upstream cluster.
func applyUpstreamTrafficSetting(upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting, upstreamCluster *xds_cluster.Cluster,
	httpProtocolOptions *extensions_upstream_http.HttpProtocolOptions) {
	// Apply Circuit Breaker threshold
	threshold := getDefaultCircuitBreakerThreshold()
	upstreamCluster.CircuitBreakers = &xds_cluster.CircuitBreakers{
		Thresholds: []*xds_cluster.CircuitBreakers_Thresholds{threshold},
	}

	if upstreamTrafficSetting == nil {
		return
	}

	connectionSettings := upstreamTrafficSetting.Spec.ConnectionSettings

	// Apply TCP connection settings
	if connectionSettings.TCP != nil {
		if connectionSettings.TCP.ConnectTimeout != nil {
			upstreamCluster.ConnectTimeout = durationpb.New(connectionSettings.TCP.ConnectTimeout.Duration)
		}
		if connectionSettings.TCP.MaxConnections != nil {
			threshold.MaxConnections = wrapperspb.UInt32(*connectionSettings.TCP.MaxConnections)
		}
	}

	// Apply HTTP connection settings
	if connectionSettings.HTTP != nil {
		if connectionSettings.HTTP.MaxRequests != nil {
			threshold.MaxRequests = wrapperspb.UInt32(*connectionSettings.HTTP.MaxRequests)
		}
		if connectionSettings.HTTP.MaxPendingRequests != nil {
			threshold.MaxPendingRequests = wrapperspb.UInt32(*connectionSettings.HTTP.MaxPendingRequests)
		}
		if connectionSettings.HTTP.MaxRetries != nil {
			threshold.MaxRetries = wrapperspb.UInt32(*connectionSettings.HTTP.MaxRetries)
		}
		if connectionSettings.HTTP.MaxRequestsPerConnection != nil {
			// TODO(#4500): When Envoy is upgraded to v1.20+, MaxRequestsPerConnection must be set
			// via the HttpProtocolOptions extensions field instead (commented below), as setting this
			// directly on the Cluster object is deprecated.
			// httpProtocolOptions.CommonHttpProtocolOptions = &xds_core.HttpProtocolOptions{
			// 	MaxRequestsPerConnection: wrapperspb.UInt32(*connectionSettings.HTTP.MaxRequestsPerConnection),
			// }
			upstreamCluster.MaxRequestsPerConnection = wrapperspb.UInt32(*connectionSettings.HTTP.MaxRequestsPerConnection)
		}
	}
}
