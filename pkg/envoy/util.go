package envoy

import (
	"strings"

	"github.com/open-service-mesh/osm/pkg/endpoint"
)

//Contains checks if a give service is in a list of services
func Contains(proxyService endpoint.NamespacedService, services []endpoint.NamespacedService) bool {
	for _, service := range services {
		if proxyService == service {
			return true
		}
	}
	return false
}

func getNamespacedService(serviceName string) endpoint.NamespacedService {
	split := strings.Split(serviceName, "/")
	var namespacedService endpoint.NamespacedService
	if len(split) == 0 {
		namespacedService.Namespace = "default"
		namespacedService.Service = split[0]
	} else {
		namespacedService.Namespace = split[0]
		namespacedService.Service = split[1]
	}
	return namespacedService
}
