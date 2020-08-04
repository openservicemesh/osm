package rotor

import (
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("certificate/CertRotor")
)

// CertRotor is a simple facility, which rotates expired certificates.
type CertRotor struct {
	certManager certificate.Manager
}
