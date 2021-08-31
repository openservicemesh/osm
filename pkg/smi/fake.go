package smi

import (
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
)

type fakeMeshSpec struct {
	trafficSplits   []*split.TrafficSplit
	httpRouteGroups []*spec.HTTPRouteGroup
	tcpRoutes       []*spec.TCPRoute
	trafficTargets  []*access.TrafficTarget
	serviceAccounts []identity.K8sServiceAccount
}

// NewFakeMeshSpecClient creates a fake Mesh Spec used for testing.
// TODO(DEPRECATE): This fake is not extendable enough, deprecate it and use mocks or re-implement fakes
func NewFakeMeshSpecClient() MeshSpec {
	return fakeMeshSpec{
		trafficSplits:   []*split.TrafficSplit{&tests.TrafficSplit},
		httpRouteGroups: []*spec.HTTPRouteGroup{&tests.HTTPRouteGroup},
		tcpRoutes:       []*spec.TCPRoute{&tests.TCPRoute},
		trafficTargets:  []*access.TrafficTarget{&tests.TrafficTarget, &tests.BookstoreV2TrafficTarget},
		serviceAccounts: []identity.K8sServiceAccount{
			tests.BookstoreServiceAccount,
			tests.BookstoreV2ServiceAccount,
			tests.BookbuyerServiceAccount,
		},
	}
}

// ListTrafficSplits lists TrafficSplit SMI resources for the fake Mesh Spec
func (f fakeMeshSpec) ListTrafficSplits(opts ...TrafficSplitListOption) []*split.TrafficSplit {
	var trafficSplits []*split.TrafficSplit
	for _, s := range f.trafficSplits {
		if filteredSplit := filterTrafficSplit(s, opts...); filteredSplit != nil {
			trafficSplits = append(trafficSplits, filteredSplit)
		}
	}
	return trafficSplits
}

// ListServiceAccounts fetches all service accounts declared with SMI Spec for the fake Mesh Spec.
func (f fakeMeshSpec) ListServiceAccounts() []identity.K8sServiceAccount {
	return f.serviceAccounts
}

// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
func (f fakeMeshSpec) ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup {
	return f.httpRouteGroups
}

// GetHTTPRouteGroup returns an SMI HTTPRouteGroup resource given its name of the form <namespace>/<name>
func (f fakeMeshSpec) GetHTTPRouteGroup(_ string) *smiSpecs.HTTPRouteGroup {
	return nil
}

// ListTCPTrafficSpecs lists SMI TCPRoute resources
func (f fakeMeshSpec) ListTCPTrafficSpecs() []*spec.TCPRoute {
	return f.tcpRoutes
}

// GetTCPRoute returns an SMI TCPRoute resource given its name of the form <namespace>/<name>s
func (f fakeMeshSpec) GetTCPRoute(_ string) *spec.TCPRoute {
	return nil
}

// ListTrafficTargets lists TrafficTarget SMI resources for the fake Mesh Spec
func (f fakeMeshSpec) ListTrafficTargets(opts ...TrafficTargetListOption) []*access.TrafficTarget {
	var trafficTargets []*access.TrafficTarget
	for _, t := range f.trafficTargets {
		if filteredTarget := filterTrafficTarget(t, opts...); filteredTarget != nil {
			trafficTargets = append(trafficTargets, filteredTarget)
		}
	}
	return trafficTargets
}
