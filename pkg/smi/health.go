package smi

import (
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	"k8s.io/client-go/discovery"
)

var (

	// requiredResourceKindGroupMap is a mapping of the required resource kind to it's API group
	requiredResourceKindGroupMap = map[string]string{
		"HTTPRouteGroup": smiSpecs.SchemeGroupVersion.String(),
		"TCPRoute":       smiSpecs.SchemeGroupVersion.String(),
		"TrafficSplit":   smiSplit.SchemeGroupVersion.String(),
		"TrafficTarget":  smiAccess.SchemeGroupVersion.String(),
	}
	smiAPIGroupVersions = []string{
		smiSpecs.SchemeGroupVersion.String(),
		smiAccess.SchemeGroupVersion.String(),
		smiSpecs.SchemeGroupVersion.String(),
		smiSplit.SchemeGroupVersion.String(),
	}
)

// HealthChecker has SMI clientset interface to access SMI CRDS
type HealthChecker struct {
	DiscoveryClient discovery.ServerResourcesInterface
}

// Liveness is the Kubernetes liveness probe handler.
func (c HealthChecker) Liveness() bool {
	return c.requiredAPIResourcesExist()
}

// Readiness is the Kubernetes readiness probe handler.
func (c HealthChecker) Readiness() bool {
	return c.requiredAPIResourcesExist()
}

// GetID returns the ID of the probe
func (c HealthChecker) GetID() string {
	return "SMI"
}

// requiredAPIResourcesExist returns true if the required API resources are available on the API server
func (c HealthChecker) requiredAPIResourcesExist() bool {
	resourceKindAvailable := make(map[string]bool)
	for resourceKind := range requiredResourceKindGroupMap {
		resourceKindAvailable[resourceKind] = false
	}
	for _, groupVersion := range smiAPIGroupVersions {
		list, err := c.DiscoveryClient.ServerResourcesForGroupVersion(groupVersion)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting resources for groupVersion %s", groupVersion)
			return false
		}
		for _, resource := range list.APIResources {
			resourceKindAvailable[resource.Kind] = true
		}
	}

	for resourceKind, isAvailable := range resourceKindAvailable {
		if !isAvailable {
			log.Error().Err(errSMICrds).Msgf("SMI API for Kind=%s, GroupVersion=%s is not available", resourceKind, requiredResourceKindGroupMap[resourceKind])
			return false
		}
	}

	return true
}
