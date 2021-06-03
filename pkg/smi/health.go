package smi

import (
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	extensionsClientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/openservicemesh/osm/pkg/version"
)

var (

	// The key is the API Resource.Kind and the value is the SMI CRD Type
	reqKinds = map[string]string{
		"HTTPRouteGroup": "specs",  // Full CRD API Group: specs.smi-spec.io
		"TCPRoute":       "specs",  // Full CRD API Group: specs.smi-spec.io
		"TrafficSplit":   "split",  // Full CRD API Group: split.smi-spec.io
		"TrafficTarget":  "access", // Full CRD API Group: access.smi-spec.io
	}
	candidateVersions = []string{smiSpecs.SchemeGroupVersion.String(), smiAccess.SchemeGroupVersion.String(), smiSpecs.SchemeGroupVersion.String(), smiSplit.SchemeGroupVersion.String()}
)

// HealthChecker has SMI clientset interface to access SMI CRDS
type HealthChecker struct {
	SMIClientset extensionsClientset.Interface
}

// Liveness is the Kubernetes liveness probe handler.
func (smi HealthChecker) Liveness() bool {
	return checkSMICrdsExist(smi.SMIClientset, reqKinds, candidateVersions)
}

// Readiness is the Kubernetes readiness probe handler.
func (smi HealthChecker) Readiness() bool {
	return checkSMICrdsExist(smi.SMIClientset, reqKinds, candidateVersions)
}

// GetID returns the ID of the probe
func (smi HealthChecker) GetID() string {
	return "SMI"
}

func checkSMICrdsExist(clientset extensionsClientset.Interface, reqKinds map[string]string, candidateVersions []string) bool {
	client := clientset.Discovery()
	for _, groupVersion := range candidateVersions {
		list, err := client.ServerResourcesForGroupVersion(groupVersion)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting resources for groupVersion %s", groupVersion)
			return false
		}
		for _, resource := range list.APIResources {
			crdName := resource.Kind
			delete(reqKinds, crdName)
		}
	}

	if len(reqKinds) != 0 {
		for missingCRD := range reqKinds {
			log.Error().Err(errSMICrds).Msgf("Missing SMI CRD: %s. To manually install %s, do `kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/%s/charts/osm/crds/%s.yaml`", missingCRD, missingCRD, version.Version, reqKinds[missingCRD])
		}
		return false
	}
	return true
}
