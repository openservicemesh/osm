package smi

import (
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	backpressure "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

type fakeMeshSpec struct {
	trafficSplits    []*split.TrafficSplit
	httpRouteGroups  []*spec.HTTPRouteGroup
	tcpRoutes        []*spec.TCPRoute
	trafficTargets   []*access.TrafficTarget
	backpressures    []*backpressure.Backpressure
	weightedServices []service.WeightedService
	serviceAccounts  []service.K8sServiceAccount
}

// NewFakeMeshSpecClient creates a fake Mesh Spec used for testing.
func NewFakeMeshSpecClient() MeshSpec {
	return fakeMeshSpec{
		trafficSplits:    []*split.TrafficSplit{&tests.TrafficSplit},
		httpRouteGroups:  []*spec.HTTPRouteGroup{&tests.HTTPRouteGroup},
		tcpRoutes:        []*spec.TCPRoute{&tests.TCPRoute},
		trafficTargets:   []*access.TrafficTarget{&tests.TrafficTarget},
		weightedServices: []service.WeightedService{tests.BookstoreV1WeightedService, tests.BookstoreV2WeightedService},
		serviceAccounts: []service.K8sServiceAccount{
			tests.BookstoreServiceAccount,
			tests.BookbuyerServiceAccount,
		},

		backpressures: []*backpressure.Backpressure{&tests.Backpressure},
	}
}

// ListTrafficSplits lists TrafficSplit SMI resources for the fake Mesh Spec.
func (f fakeMeshSpec) ListTrafficSplits() []*split.TrafficSplit {
	return f.trafficSplits
}

// ListTrafficSplitServices fetches all services declared with SMI Spec for the fake Mesh Spec.
func (f fakeMeshSpec) ListTrafficSplitServices() []service.WeightedService {
	return f.weightedServices
}

// ListServiceAccounts fetches all service accounts declared with SMI Spec for the fake Mesh Spec.
func (f fakeMeshSpec) ListServiceAccounts() []service.K8sServiceAccount {
	return f.serviceAccounts
}

// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
func (f fakeMeshSpec) ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup {
	return f.httpRouteGroups
}

// ListTCPTrafficSpecs lists SMI TCPRoute resources
func (f fakeMeshSpec) ListTCPTrafficSpecs() []*spec.TCPRoute {
	return f.tcpRoutes
}

// GetTCPRoute returns an SMI TCPRoute resource given its name of the form <namespace>/<name>s
func (f fakeMeshSpec) GetTCPRoute(_ string) *spec.TCPRoute {
	return nil
}

// ListTrafficTargets lists TrafficTarget SMI resources for the fake Mesh Spec.
func (f fakeMeshSpec) ListTrafficTargets() []*access.TrafficTarget {
	return f.trafficTargets
}

func (f fakeMeshSpec) GetBackpressurePolicy(_ service.MeshService) *backpressure.Backpressure {
	return nil
}

// GetAnnouncementsChannel returns the channel on which SMI makes announcements for the fake Mesh Spec.
func (f fakeMeshSpec) GetAnnouncementsChannel() <-chan announcements.Announcement {
	return make(chan announcements.Announcement)
}
