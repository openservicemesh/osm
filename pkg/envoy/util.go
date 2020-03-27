package envoy

import (
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
