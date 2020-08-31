package smi

import (
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha3"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"

	backpressure "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

type fakeMeshSpec struct {
	trafficSplits    []*split.TrafficSplit
	httpRouteGroups  []*spec.HTTPRouteGroup
	tcpRoutes        []*spec.TCPRoute
	trafficTargets   []*target.TrafficTarget
	backpressures    []*backpressure.Backpressure
	weightedServices []service.WeightedService
	serviceAccounts  []service.K8sServiceAccount
	services         []*corev1.Service
}

// NewFakeMeshSpecClient creates a fake Mesh Spec used for testing.
func NewFakeMeshSpecClient() MeshSpec {
	return fakeMeshSpec{
		trafficSplits:    []*split.TrafficSplit{&tests.TrafficSplit},
		httpRouteGroups:  []*spec.HTTPRouteGroup{&tests.HTTPRouteGroup},
		tcpRoutes:        []*spec.TCPRoute{&tests.TCPRoute},
		trafficTargets:   []*target.TrafficTarget{&tests.TrafficTarget},
		weightedServices: []service.WeightedService{tests.WeightedService},
		serviceAccounts: []service.K8sServiceAccount{
			tests.BookstoreServiceAccount,
			tests.BookbuyerServiceAccount,
		},
		services: []*corev1.Service{
			tests.NewServiceFixture(tests.BookstoreService.Name, tests.BookstoreService.Namespace, nil),
			tests.NewServiceFixture(tests.BookbuyerService.Name, tests.BookbuyerService.Namespace, nil),
			tests.NewServiceFixture(tests.BookstoreApexService.Name, tests.BookstoreApexService.Namespace, nil),
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

// GetService fetches a specific service declared in SMI for the fake Mesh Spec.
func (f fakeMeshSpec) GetService(svc service.MeshService) *corev1.Service {
	for _, service := range f.services {
		if service.Name == svc.Name && service.Namespace == svc.Namespace {
			return service
		}
	}
	return nil
}

// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
func (f fakeMeshSpec) ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup {
	return f.httpRouteGroups
}

// ListTCPTrafficSpecs lists SMI TCPRoute resources
func (f fakeMeshSpec) ListTCPTrafficSpecs() []*spec.TCPRoute {
	return f.tcpRoutes
}

// ListTrafficTargets lists TrafficTarget SMI resources for the fake Mesh Spec.
func (f fakeMeshSpec) ListTrafficTargets() []*target.TrafficTarget {
	return f.trafficTargets
}

func (f fakeMeshSpec) GetBackpressurePolicy(svc service.MeshService) *backpressure.Backpressure {
	return nil
}

// GetAnnouncementsChannel returns the channel on which SMI makes announcements for the fake Mesh Spec.
func (f fakeMeshSpec) GetAnnouncementsChannel() <-chan interface{} {
	return make(chan interface{})
}

// ListServices returns a list of services that are part of monitored namespaces
func (f fakeMeshSpec) ListServices() []*corev1.Service {
	return f.services
}
