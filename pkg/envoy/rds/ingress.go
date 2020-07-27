package rds

import (
	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/trafficpolicy"
)

func updateRoutesForIngress(svc service.NamespacedService, catalog catalog.MeshCataloger, routesPerHost map[string]map[string]trafficpolicy.RouteWeightedClusters) error {
	ingressRoutesPerHost, err := catalog.GetIngressRoutesPerHost(svc)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress route configuration for proxy %s", svc)
		return err
	}

	if len(ingressRoutesPerHost) == 0 {
		return nil
	}

	ingressWeightedCluster := service.WeightedCluster{
		ClusterName: service.ClusterName(svc.String()),
		Weight:      constants.ClusterWeightAcceptAll,
	}

	for host, routes := range ingressRoutesPerHost {
		for _, rt := range routes {
			aggregateRoutesByHost(routesPerHost, rt, ingressWeightedCluster, host)
		}
	}

	log.Trace().Msgf("Ingress routes for service %s: %+v", svc, routesPerHost)

	return nil
}
