package smi

import (
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	extensionsClientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/openservicemesh/osm/pkg/version"
)

// HealthChecker has SMI clientset interface to access SMI CRDS
type HealthChecker struct {
	SMIClientset extensionsClientset.Interface
}

// Liveness is the Kubernetes liveness probe handler.
func (smi HealthChecker) Liveness() bool {
	return true
}

// Readiness is the Kubernetes readiness probe handler.
func (smi HealthChecker) Readiness() bool {
	return checkSMICrdsExist(smi.SMIClientset)
}

// GetID returns the ID of the probe
func (smi HealthChecker) GetID() string {
	return "SMI"
}

func checkSMICrdsExist(clientset extensionsClientset.Interface) bool {
	client := clientset.Discovery()
	// The key is the API Resource.Kind
	// The value is the SMI CRD Type
	reqVersions := map[string]string{
		"HTTPRouteGroup": "specs",  // Full CRD API Group: specs.smi-spec.io
		"TCPRoute":       "specs",  // Full CRD API Group: specs.smi-spec.io
		"TrafficSplit":   "split",  // Full CRD API Group: split.smi-spec.io
		"TrafficTarget":  "access", // Full CRD API Group: access.smi-spec.io
	}
	var candidateVersions = []string{smiAccess.SchemeGroupVersion.String(), smiSpecs.SchemeGroupVersion.String(), smiSplit.SchemeGroupVersion.String()}

	for _, groupVersion := range candidateVersions {
		list, err := client.ServerResourcesForGroupVersion(groupVersion)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting resources for groupVersion %s", groupVersion)
			return false
		}
		for _, resource := range list.APIResources {
			crdName := resource.Kind
			delete(reqVersions, crdName)
		}
	}

	if len(reqVersions) != 0 {
		for missingCRD := range reqVersions {
			log.Error().Err(errSMICrds).Msgf("Missing SMI CRD: %s. To manually install %s, do `kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/%s/charts/osm/crds/%s.yaml`", missingCRD, missingCRD, version.Version, reqVersions[missingCRD])
		}
		return false
	}
	return true
}
