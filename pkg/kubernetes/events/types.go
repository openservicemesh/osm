package events

import (
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
