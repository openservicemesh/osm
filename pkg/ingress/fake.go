package ingress

import (
	extensionsV1beta "k8s.io/api/extensions/v1beta1"

	"github.com/open-service-mesh/osm/pkg/endpoint"
)

type fakeIngressMonitor struct{}

// NewFakeIngressMonitor returns a fake ingress.Monitor used for testing
func NewFakeIngressMonitor() Monitor {
	return fakeIngressMonitor{}
}

// GetIngressResources returns the ingress resources whose backends correspond to the service
func (f fakeIngressMonitor) GetIngressResources(endpoint.NamespacedService) ([]*extensionsV1beta.Ingress, error) {
	return nil, nil
}

// GetAnnouncementsChannel returns the channel on which Ingress Monitor makes annoucements
func (f fakeIngressMonitor) GetAnnouncementsChannel() <-chan interface{} {
	return make(chan interface{})
}
