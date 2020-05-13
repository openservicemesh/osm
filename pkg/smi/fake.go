package smi

import (
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha1"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/tests"
)

type fakeMeshSpec struct {
	routeGroups      []*spec.HTTPRouteGroup
	trafficTargets   []*target.TrafficTarget
	weightedServices []service.WeightedService
	serviceAccounts  []service.NamespacedServiceAccount
}

// NewFakeMeshSpecClient creates a fake Mesh Spec used for testing.
func NewFakeMeshSpecClient() MeshSpec {
	return fakeMeshSpec{
		routeGroups:      []*spec.HTTPRouteGroup{&tests.HTTPRouteGroup},
		trafficTargets:   []*target.TrafficTarget{&tests.TrafficTarget},
		weightedServices: []service.WeightedService{tests.WeightedService},
		serviceAccounts: []service.NamespacedServiceAccount{
			tests.BookstoreServiceAccount,
			tests.BookbuyerServiceAccount,
		},
	}
}

// ListTrafficSplits lists TrafficSplit SMI resources for the fake Mesh Spec.
func (f fakeMeshSpec) ListTrafficSplits() []*split.TrafficSplit {
	return nil
}

// ListServices fetches all services declared with SMI Spec for the fake Mesh Spec.
func (f fakeMeshSpec) ListServices() []service.WeightedService {
	return f.weightedServices
}

// ListServiceAccounts fetches all service accounts declared with SMI Spec for the fake Mesh Spec.
func (f fakeMeshSpec) ListServiceAccounts() []service.NamespacedServiceAccount {
	return f.serviceAccounts
}

// GetService fetches a specific service declared in SMI for the fake Mesh Spec.
func (f fakeMeshSpec) GetService(service.Name) (service *corev1.Service, exists bool, err error) {
	return nil, false, nil
}

// ListHTTPTrafficSpecs lists TrafficSpec SMI resources for the fake Mesh Spec.
func (f fakeMeshSpec) ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup {
	return f.routeGroups
}

// ListTrafficTargets lists TrafficTarget SMI resources for the fake Mesh Spec.
func (f fakeMeshSpec) ListTrafficTargets() []*target.TrafficTarget {
	return f.trafficTargets
}

// GetAnnouncementsChannel returns the channel on which SMI makes announcements for the fake Mesh Spec.
func (f fakeMeshSpec) GetAnnouncementsChannel() <-chan interface{} {
	return make(chan interface{})
}
