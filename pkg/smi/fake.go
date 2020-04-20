package smi

import (
	"github.com/open-service-mesh/osm/pkg/endpoint"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha1"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
)

type fakeMeshSpec struct{}

// NewFakeMeshSpecClient creates a fake Mesh Spec used for testing.
func NewFakeMeshSpecClient() MeshSpec {
	return fakeMeshSpec{}
}

// ListTrafficSplits lists TrafficSplit SMI resources for the fake Mesh Spec.
func (f fakeMeshSpec) ListTrafficSplits() []*split.TrafficSplit {
	return nil
}

// ListServices fetches all services declared with SMI Spec for the fake Mesh Spec.
func (f fakeMeshSpec) ListServices() []endpoint.WeightedService {
	return nil
}

// ListServiceAccounts fetches all service accounts declared with SMI Spec for the fake Mesh Spec.
func (f fakeMeshSpec) ListServiceAccounts() []endpoint.NamespacedServiceAccount {
	return nil
}

// GetService fetches a specific service declared in SMI for the fake Mesh Spec.
func (f fakeMeshSpec) GetService(endpoint.ServiceName) (service *corev1.Service, exists bool, err error) {
	return nil, false, nil
}

// ListHTTPTrafficSpecs lists TrafficSpec SMI resources for the fake Mesh Spec.
func (f fakeMeshSpec) ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup {
	return nil
}

// ListTrafficTargets lists TrafficTarget SMI resources for the fake Mesh Spec.
func (f fakeMeshSpec) ListTrafficTargets() []*target.TrafficTarget {
	return nil
}

// GetAnnouncementsChannel returns the channel on which SMI makes announcements for the fake Mesh Spec.
func (f fakeMeshSpec) GetAnnouncementsChannel() <-chan interface{} {
	return make(chan interface{})
}
