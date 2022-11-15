package kube

import (
	corev1 "k8s.io/api/core/v1"
)

// GetTargetPortFromEndpoints returns the endpoint port corresponding to the given endpoint name and endpoints
// TODO(4863): unexport this method, it should not be used outside of this package.
func GetTargetPortFromEndpoints(endpointName string, endpoints corev1.Endpoints) (endpointPort uint16) {
	// Per https://pkg.go.dev/k8s.io/api/core/v1#ServicePort and
	// https://pkg.go.dev/k8s.io/api/core/v1#EndpointPort, if a service has multiple
	// ports, then ServicePort.Name must match EndpointPort.Name when considering
	// matching endpoints for the service's port. ServicePort.Name and EndpointPort.Name
	// can be unset when the service has a single port exposed, in which case we are
	// guaranteed to have the same port specified in the list of EndpointPort.Subsets.
	//
	// The logic below works as follows:
	// If the service has multiple ports, retrieve the matching endpoint port using
	// the given ServicePort.Name specified by `endpointName`.
	// Otherwise, simply return the only port referenced in EndpointPort.Subsets.
	for _, subset := range endpoints.Subsets {
		if endpointName == "" || len(subset.Ports) == 1 {
			// ServicePort.Name is not passed or a single port exists on the service.
			// Both imply that this service has a single ServicePort and EndpointPort.
			endpointPort = uint16(subset.Ports[0].Port)
			return
		}
		for _, port := range subset.Ports {
			// If more than 1 port is specified
			if port.Name == endpointName {
				endpointPort = uint16(port.Port)
				return
			}
		}
	}
	return
}
