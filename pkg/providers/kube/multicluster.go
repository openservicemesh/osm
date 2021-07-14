package kube

import (
	"net"
	"strconv"
	"strings"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/endpoint"
)

const portIPSeparator = `:`

func (c *Client) getMultiClusterServiceEndpointsForServiceAccount(serviceAccount, namespace string) []endpoint.Endpoint {
	multiclusterServices := c.configClient.GetMultiClusterServiceByServiceAccount(serviceAccount, namespace)
	log.Trace().Msgf("Multicluster services for service account %s: %+v", serviceAccount, multiclusterServices)
	return getEndpointsFromMultiClusterServices(multiclusterServices...)
}

func getEndpointsFromMultiClusterServices(services ...v1alpha1.MultiClusterService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	for _, svc := range services {
		log.Trace().Msgf("Working on service: %s --> spec=%+v", svc, svc.Spec.Clusters)
		for _, cluster := range svc.Spec.Clusters {
			log.Trace().Msgf("Looking for IP and Port for cluster=%s for service %s", cluster.Name, svc)
			if ip, port, err := getIPPort(cluster, svc); err != nil {
				log.Err(err).Msgf("Error getting IP and Port for cluster=%s for service %s", cluster.Name, svc)
			} else {
				endpoints = append(endpoints, endpoint.Endpoint{
					IP:   net.ParseIP(ip),
					Port: endpoint.Port(port),
				})
			}
		}
	}

	log.Trace().Msgf("Multicluster endpoints for service %+v: %+v", services, endpoints)
	return endpoints
}

func getIPPort(cluster v1alpha1.ClusterSpec, svc v1alpha1.MultiClusterService) (ip string, port int, err error) {
	tokens := strings.Split(cluster.Address, portIPSeparator)
	if len(tokens) != 2 {
		log.Error().Msgf("Error parsing remote service %s address %s. It should have IP address and port number", svc.Name, cluster.Address)
		return "", 0, err
	}

	ip, portStr := tokens[0], tokens[1]
	port, err = strconv.Atoi(portStr)
	if err != nil {
		log.Error().Msgf("Remote service %s port number format invalid. Remote cluster address: %s", svc.Name, cluster.Address)
		return "", 0, err
	}

	return ip, port, err
}
