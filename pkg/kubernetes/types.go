package kubernetes

import (
	"time"

	"github.com/open-service-mesh/osm/pkg/logger"
)

var (
	log = logger.New("kube-events")
)

// EventType is the type of event we have received from Kubernetes
type EventType int

const (
	// CreateEvent is a type of a Kubernetes API event.
	CreateEvent EventType = iota + 1

	// UpdateEvent is a type of a Kubernetes API event.
	UpdateEvent

	// DeleteEvent is a type of a Kubernetes API event.
	DeleteEvent
)

const (
	// DefaultKubeEventResyncInterval is the default resync interval for k8s events
	DefaultKubeEventResyncInterval = 30 * time.Second
)

// Event is the combined type and actual object we received from Kubernetes
type Event struct {
	Type  EventType
	Value interface{}
}
