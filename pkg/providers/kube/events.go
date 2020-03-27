package kube

// EventType is the type of event we have received from Kubernetes
type EventType int

const (
	// Create is a type of a Kubernetes API event.
	Create EventType = iota + 1

	// Update is a type of a Kubernetes API event.
	Update

	// Delete is a type of a Kubernetes API event.
	Delete
)

// Event is the combined type and actual object we received from Kubernetes
type Event struct {
	Type  EventType
	Value interface{}
}
