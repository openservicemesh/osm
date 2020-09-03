package namespace

import (
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("kube-namespace")
)

// Controller is the controller interface for K8s namespaces
type Controller interface {
	// IsMonitoredNamespace returns whether a namespace with the given name is being monitored
	// by the mesh
	IsMonitoredNamespace(string) bool

	// ListMonitoredNamespaces returns the namespaces monitored by the mesh
	ListMonitoredNamespaces() ([]string, error)

	// GetAnnouncementsChannel returns the channel on which namespace makes announcements
	GetAnnouncementsChannel() <-chan interface{}
}
