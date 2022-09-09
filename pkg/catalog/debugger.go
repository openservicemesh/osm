package catalog

import (
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"github.com/openservicemesh/osm/pkg/identity"
)

// ListSMIPolicies returns all policies OSM is aware of.
func (mc *MeshCatalog) ListSMIPolicies() ([]*split.TrafficSplit, []identity.K8sServiceAccount, []*spec.HTTPRouteGroup, []*access.TrafficTarget) {
	trafficSplits := mc.Interface.ListTrafficSplits()
	serviceAccounts := mc.Interface.ListServiceAccountsFromTrafficTargets()
	trafficSpecs := mc.Interface.ListHTTPTrafficSpecs()
	trafficTargets := mc.Interface.ListTrafficTargets()

	return trafficSplits, serviceAccounts, trafficSpecs, trafficTargets
}
