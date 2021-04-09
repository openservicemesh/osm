package smi

import (
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

type fakeMeshSpec struct {
	trafficSplits   []*split.TrafficSplit
	httpRouteGroups []*spec.HTTPRouteGroup
	tcpRoutes       []*spec.TCPRoute
	trafficTargets  []*access.TrafficTarget
	serviceAccounts []service.K8sServiceAccount
}

// NewFakeMeshSpecClient creates a fake Mesh Spec used for testing.
func NewFakeMeshSpecClient() MeshSpec {
	return fakeMeshSpec{
		trafficSplits:   []*split.TrafficSplit{&tests.TrafficSplit},
		httpRouteGroups: []*spec.HTTPRouteGroup{&tests.HTTPRouteGroup},
		tcpRoutes:       []*spec.TCPRoute{&tests.TCPRoute},
		trafficTargets:  []*access.TrafficTarget{&tests.TrafficTarget, &tests.BookstoreV2TrafficTarget},
		serviceAccounts: []service.K8sServiceAccount{
			tests.BookstoreServiceAccount,
			tests.BookstoreV2ServiceAccount,
			tests.BookbuyerServiceAccount,
		},
	}
}

// ListTrafficSplits lists TrafficSplit SMI resources for the fake Mesh Spec.
func (f fakeMeshSpec) ListTrafficSplits() []*split.TrafficSplit {
	return f.trafficSplits
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

// GetAnnouncementsChannel returns the channel on which SMI makes announcements for the fake Mesh Spec.
func (f fakeMeshSpec) GetAnnouncementsChannel() <-chan announcements.Announcement {
	return make(chan announcements.Announcement)
}
