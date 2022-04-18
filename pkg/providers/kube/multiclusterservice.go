package kube

import (
	"net"
	"strconv"
	"strings"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
)

const portIPSeparator = `:`

// getMulticlusterEndpoints returns the endpoints for multicluster services if such exist.
func (c *client) getMulticlusterEndpoints(svc service.MeshService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	serviceIdentities, err := c.kubeController.ListServiceIdentitiesForService(svc)
	if err != nil {
		log.Error().Str(constants.LogFieldContext, constants.LogContextMulticluster).Err(err).Msgf("[%s] Error getting Multicluster service identities for service %s", c.GetID(), svc.Name)
		return endpoints
	}

	for _, ident := range serviceIdentities {
		remoteEndpoints := c.getMultiClusterServiceEndpointsForServiceAccount(ident.Name, ident.Namespace)
		endpoints = append(endpoints, remoteEndpoints...)
	}

	log.Debug().Str(constants.LogFieldContext, constants.LogContextMulticluster).Msgf("[%s] Multicluster Endpoints for service %s: %+v", c.GetID(), svc, endpoints)
	return endpoints
}

// getMultiClusterServiceEndpointsForServiceAccount returns the multicluster services for a service account if such exist.
func (c *client) getMultiClusterServiceEndpointsForServiceAccount(serviceAccount, namespace string) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	services := c.configClient.GetMultiClusterServiceByServiceAccount(serviceAccount, namespace)
	if len(services) <= 0 {
		return endpoints
	}

	for _, svc := range services {
		for _, cluster := range svc.Spec.Clusters {
			ip, port, err := getIPPort(cluster)
			if err != nil {
				log.Error().Err(err).Str(constants.LogFieldContext, constants.LogContextMulticluster).Msgf("Error getting IP and Port for cluster=%s for service %s", cluster.Name, svc)
				continue
			}

			ep := endpoint.Endpoint{
				IP:       ip,
				Port:     endpoint.Port(port),
				Weight:   endpoint.Weight(cluster.Weight),
				Priority: endpoint.Priority(cluster.Priority),
				Zone:     cluster.Name,
			}
			endpoints = append(endpoints, ep)
		}
	}
	log.Debug().Str(constants.LogFieldContext, constants.LogContextMulticluster).Msgf("[%s] Multicluster Endpoints for service account %s: %+v", c.GetID(), serviceAccount, endpoints)
	return endpoints
}

func getIPPort(cluster configv1alpha3.ClusterSpec) (ip net.IP, port int, err error) {
	tokens := strings.Split(cluster.Address, portIPSeparator)
	if len(tokens) != 2 {
		log.Error().Err(errParseMulticlusterServiceIP).Str(constants.LogFieldContext, constants.LogContextMulticluster).Msgf("Invalid address format %s. It should have IP address and port number separated by %s", cluster.Address, portIPSeparator)
		return nil, 0, errParseMulticlusterServiceIP
	}

	ipStr, portStr := tokens[0], tokens[1]
	port, err = strconv.Atoi(portStr)
	if err != nil {
		log.Error().Str(constants.LogFieldContext, constants.LogContextMulticluster).Msgf("Invalid port number format %s for cluster address: %s", portStr, cluster.Address)
		return nil, 0, err
	}

	ip = net.ParseIP(ipStr)
	return ip, port, err
}
