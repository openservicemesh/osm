package catalog

import (
	"fmt"
	"time"

	envoyV2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	protobufTypes "github.com/gogo/protobuf/types"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy/rc"
	"github.com/deislabs/smc/pkg/smi"
)

const (
	//HTTPTraffic specifies HTTP Traffic Policy
	HTTPTraffic = "HTTPRouteGroup"
)

// NewServiceCatalog creates a new service catalog
func NewServiceCatalog(meshSpec smi.MeshSpec, endpointsProviders ...endpoint.Provider) MeshCataloger {
	glog.Info("[catalog] Create a new Service Catalog.")
	serviceCatalog := MeshCatalog{
		servicesCache:      make(map[endpoint.ServiceName][]endpoint.Endpoint),
		endpointsProviders: endpointsProviders,
		meshSpec:           meshSpec,
	}

	// NOTE(draychev): helpful while developing alpha MVP -- remove before releasing beta version.
	go func() {
		counter := 0
		for {
			glog.V(7).Infof("------------------------- Service Catalog Cache Refresh %d -------------------------", counter)
			counter++
			serviceCatalog.refreshCache()
			time.Sleep(5 * time.Second)
		}
	}()
	return &serviceCatalog
}

// ListTrafficRoutes constructs a DiscoveryResponse with all routes the given Envoy proxy should be aware of.
// The bool return value indicates whether there have been any changes since the last invocation of this function.
func (sc *MeshCatalog) ListTrafficRoutes(clientID smi.ClientIdentity) (*envoyV2.DiscoveryResponse, bool, error) {
	glog.Info("[catalog] Listing Routes for client: ", clientID)
	allRoutes, err := sc.getHTTPPathsPerRoute()
	if err != nil {
		glog.Error("[catalog] Could not get all routes: ", err)
		return nil, false, err
	}

	allTrafficPolicies, err := getTrafficPolicyPerRoute(sc, allRoutes)
	if err != nil {
		glog.Error("[catalog] Could not get all traffic policies: ", err)
		return nil, false, err
	}

	var protos []*protobufTypes.Any
	for _, trafficPolicies := range allTrafficPolicies {
		routeConfiguration := rc.NewRouteConfiguration(trafficPolicies)

		proto, err := protobufTypes.MarshalAny(&routeConfiguration)
		if err != nil {
			glog.Errorf("[catalog] Error marshalling RouteConfigurationURI %+v: %s", routeConfiguration, err)
			continue
		}
		protos = append(protos, proto)
	}

	resp := &envoyV2.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   rc.RouteConfigurationURI,
	}

	return resp, false, nil
}

func (sc *MeshCatalog) getHTTPPathsPerRoute() (map[string]endpoint.RoutePaths, error) {
	routes := make(map[string]endpoint.RoutePaths)
	for _, trafficSpecs := range sc.meshSpec.ListHTTPTrafficSpecs() {
		glog.V(7).Infof("[RDS][catalog] Discovered TrafficSpec resource: %s/%s \n", trafficSpecs.Namespace, trafficSpecs.Name)
		if trafficSpecs.Matches == nil {
			glog.Errorf("[RDS][catalog] TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}
		trafficKind := trafficSpecs.Kind
		spec := fmt.Sprintf("%s/%s/%s", trafficSpecs.Name, trafficKind, trafficSpecs.Namespace)
		//todo (snchh) : no mapping yet for route methods (GET,POST) in the envoy configuration
		for _, trafficSpecsMatches := range trafficSpecs.Matches {
			serviceRoute := endpoint.RoutePaths{}
			serviceRoute.RoutePathRegex = trafficSpecsMatches.PathRegex
			serviceRoute.RouteMethods = trafficSpecsMatches.Methods
			routes[fmt.Sprintf("%s/%s", spec, trafficSpecsMatches.Name)] = serviceRoute
		}
	}
	glog.V(7).Infof("[catalog] Constructed HTTP path routes: %+v", routes)
	return routes, nil
}

func getTrafficPolicyPerRoute(sc *MeshCatalog, routes map[string]endpoint.RoutePaths) ([]endpoint.TrafficTargetPolicies, error) {
	var trafficPolicies []endpoint.TrafficTargetPolicies
	for _, trafficTargets := range sc.meshSpec.ListTrafficTargets() {
		glog.V(7).Infof("[RDS][catalog] Discovered TrafficTarget resource: %s/%s \n", trafficTargets.Namespace, trafficTargets.Name)
		if trafficTargets.Specs == nil || len(trafficTargets.Specs) == 0 {
			glog.Errorf("[RDS][catalog] TrafficTarget %s/%s has no spec routes; Skipping...", trafficTargets.Namespace, trafficTargets.Name)
			continue
		}

		for _, trafficSources := range trafficTargets.Sources {
			trafficTargetPolicy := endpoint.TrafficTargetPolicies{}
			trafficTargetPolicy.PolicyName = trafficTargets.Name
			trafficTargetPolicy.Destination = trafficTargets.Destination.Name
			trafficTargetPolicy.Source = trafficSources.Name
			for _, trafficTargetSpecs := range trafficTargets.Specs {
				if trafficTargetSpecs.Kind != HTTPTraffic {
					glog.Errorf("[RDS][catalog] TrafficTarget %s/%s has Spec Kind %s which isn't supported for now; Skipping...", trafficTargets.Namespace, trafficTargets.Name, trafficTargetSpecs.Kind)
					continue
				}
				trafficTargetPolicy.PolicyRoutePaths = []endpoint.RoutePaths{}

				for _, specMatches := range trafficTargetSpecs.Matches {
					routePath := routes[fmt.Sprintf("%s/%s/%s/%s", trafficTargetSpecs.Name, trafficTargetSpecs.Kind, trafficTargets.Namespace, specMatches)]
					trafficTargetPolicy.PolicyRoutePaths = append(trafficTargetPolicy.PolicyRoutePaths, routePath)
				}
			}
			trafficPolicies = append(trafficPolicies, trafficTargetPolicy)
		}
	}

	glog.V(7).Infof("[catalog] Constructed traffic routes: %+v", trafficPolicies)
	return trafficPolicies, nil
}

// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
func (sc *MeshCatalog) RegisterNewEndpoint(smi.ClientIdentity) {
	// TODO(draychev): implement
	panic("NotImplemented")
}

// ListEndpointsProviders retrieves the full list of endpoints providers registered with Service Catalog so far.
func (sc *MeshCatalog) ListEndpointsProviders() []endpoint.Provider {
	// TODO(draychev): implement
	panic("NotImplemented")
}
