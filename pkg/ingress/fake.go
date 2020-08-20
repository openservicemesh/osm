package ingress

import (
	extensionsV1beta "k8s.io/api/extensions/v1beta1"

	"github.com/openservicemesh/osm/pkg/service"
)

// FakeIngressMonitor returns a fake ingress monitor object
type FakeIngressMonitor struct {
	FakeIngresses []*extensionsV1beta.Ingress
	Monitor
}

// NewFakeIngressMonitor returns a fake ingress.Monitor used for testing
func NewFakeIngressMonitor() FakeIngressMonitor {
	return FakeIngressMonitor{}
}

// GetIngressResources returns the ingress resources whose backends correspond to the service
func (f FakeIngressMonitor) GetIngressResources(service.MeshService) ([]*extensionsV1beta.Ingress, error) {
	return f.FakeIngresses, nil
}

// GetAnnouncementsChannel returns the channel on which Ingress Monitor makes annoucements
func (f FakeIngressMonitor) GetAnnouncementsChannel() <-chan interface{} {
	return make(chan interface{})
}
