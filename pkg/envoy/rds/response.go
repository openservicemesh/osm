package rds

import (
	set "github.com/deckarep/golang-set"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// NewResponse creates a new Route Discovery Response.
func NewResponse(catalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, _ configurator.Configurator, _ certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	svcList, err := catalog.GetServicesFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	// Github Issue #1575
	proxyServiceName := svcList[0]

	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msg("Failed listing routes")
		return nil, err
	}
	log.Debug().Msgf("trafficPolicies: %+v", allTrafficPolicies)

	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeRDS),
	}

	allTrafficSplits, _, _, _, _ := catalog.ListSMIPolicies()
	var routeConfiguration []*xds_route.RouteConfiguration
	outboundRouteConfig := route.NewRouteConfigurationStub(route.OutboundRouteConfigName)
	inboundRouteConfig := route.NewRouteConfigurationStub(route.InboundRouteConfigName)
	outboundAggregatedRoutesByHostnames := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)
	inboundAggregatedRoutesByHostnames := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)

	for _, trafficPolicy := range allTrafficPolicies {
		isSourceService := trafficPolicy.Source.Equals(proxyServiceName)
		isDestinationService := trafficPolicy.Destination.Equals(proxyServiceName)
		svc := trafficPolicy.Destination
		hostnames, err := catalog.GetResolvableHostnamesForUpstreamService(proxyServiceName, svc)
		//filter out traffic split service, reference to pkg/catalog/xds_certificates.go:74
		if isTrafficSplitService(svc, allTrafficSplits) {
			continue
		}
		if err != nil {
			log.Error().Err(err).Msg("Failed listing domains")
			return nil, err
		}
		weightedCluster, err := catalog.GetWeightedClusterForService(svc)
		if err != nil {
			log.Error().Err(err).Msg("Failed listing clusters")
			return nil, err
		}

		for _, hostname := range hostnames {
			// All routes from a given source to destination are part of 1 traffic policy between the source and destination.
			for _, httpRoute := range trafficPolicy.HTTPRouteMatches {
				if isSourceService {
					aggregateRoutesByHost(outboundAggregatedRoutesByHostnames, httpRoute, weightedCluster, hostname)
				}

				if isDestinationService {
					aggregateRoutesByHost(inboundAggregatedRoutesByHostnames, httpRoute, weightedCluster, hostname)
				}
			}
		}
	}

	if err = updateRoutesForIngress(proxyServiceName, catalog, inboundAggregatedRoutesByHostnames); err != nil {
		return nil, err
	}

	route.UpdateRouteConfiguration(outboundAggregatedRoutesByHostnames, outboundRouteConfig, route.OutboundRoute)
	route.UpdateRouteConfiguration(inboundAggregatedRoutesByHostnames, inboundRouteConfig, route.InboundRoute)
	routeConfiguration = append(routeConfiguration, outboundRouteConfig)
	routeConfiguration = append(routeConfiguration, inboundRouteConfig)

	for _, config := range routeConfiguration {
		marshalledRouteConfig, err := ptypes.MarshalAny(config)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal route config for proxy")
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledRouteConfig)
	}
	return resp, nil
}

func isTrafficSplitService(svc service.MeshService, allTrafficSplits []*split.TrafficSplit) bool {
	for _, split := range allTrafficSplits {
		if split.Namespace == svc.Namespace && split.Spec.Service == svc.Name {
			return true
		}
	}
	return false
}

func aggregateRoutesByHost(routesPerHost map[string]map[string]trafficpolicy.RouteWeightedClusters, routePolicy trafficpolicy.HTTPRouteMatch, weightedCluster service.WeightedCluster, hostname string) {
	host := kubernetes.GetServiceFromHostname(hostname)
	_, exists := routesPerHost[host]
	if !exists {
		// no host found, create a new route map
		routesPerHost[host] = make(map[string]trafficpolicy.RouteWeightedClusters)
	}
	routePolicyWeightedCluster, routeFound := routesPerHost[host][routePolicy.PathRegex]
	if routeFound {
		// add the cluster to the existing route
		routePolicyWeightedCluster.WeightedClusters.Add(weightedCluster)
		routePolicyWeightedCluster.HTTPRouteMatch.Methods = append(routePolicyWeightedCluster.HTTPRouteMatch.Methods, routePolicy.Methods...)
		if routePolicyWeightedCluster.HTTPRouteMatch.Headers == nil {
			routePolicyWeightedCluster.HTTPRouteMatch.Headers = make(map[string]string)
		}
		for headerKey, headerValue := range routePolicy.Headers {
			routePolicyWeightedCluster.HTTPRouteMatch.Headers[headerKey] = headerValue
		}
		routePolicyWeightedCluster.Hostnames.Add(hostname)
		routesPerHost[host][routePolicy.PathRegex] = routePolicyWeightedCluster
	} else {
		// no route found, create a new route and cluster mapping on host
		routesPerHost[host][routePolicy.PathRegex] = createRoutePolicyWeightedClusters(routePolicy, weightedCluster, hostname)
	}
}

func createRoutePolicyWeightedClusters(routePolicy trafficpolicy.HTTPRouteMatch, weightedCluster service.WeightedCluster, hostname string) trafficpolicy.RouteWeightedClusters {
	return trafficpolicy.RouteWeightedClusters{
		HTTPRouteMatch:   routePolicy,
		WeightedClusters: set.NewSet(weightedCluster),
		Hostnames:        set.NewSet(hostname),
	}
}
