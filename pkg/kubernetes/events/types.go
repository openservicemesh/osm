package events

import (
	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("kube-events")
)

// Kubernetes Fatal Event reasons
// Fatal events are prefixed with 'Fatal' to help the event recording framework to wait for fatal
// events to be recorded prior to aborting.
const (
	// InvalidCLIParameters signifies invalid CLI parameters
	InvalidCLIParameters = "FatalInvalidCLIParameters"

	// InitializationError signifies an error during initialization
	InitializationError = "FatalInitializationError"

	// InvalidCertificateManager signifies that the certificate manager is invalid
	InvalidCertificateManager = "FatalInvalidCertificateManager"

	// CertificateIssuanceFailure signifies that a request to issue a certificate failed
	CertificateIssuanceFailure = "FatalCertificateIssuanceFailure"
)

// PubSubMessage represents a common messages abstraction to pass through the PubSub interface
type PubSubMessage struct {
	AnnouncementType announcements.AnnouncementType
	OldObj           interface{}
	NewObj           interface{}
}

// PubSub is a simple interface to call for pubsub functionality in front of a pubsub implementation
type PubSub interface {
	// Subscribe returns a channel subscribed to the specific type/s of announcement/s passed by parameter
	Subscribe(aTypes ...announcements.AnnouncementType) chan interface{}

	// Publish publishes the message to all subscribers that have subscribed to <message.AnnouncementType> topic
	Publish(message PubSubMessage)

	// Unsub unsubscribes and closes the channel on pubsub backend
	// Note this is a necessary step to ensure a channel can be
	// garbage collected when it is freed.
	Unsub(unsubChan chan interface{})
}
