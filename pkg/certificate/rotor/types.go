package rotor

import (
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("certificate/rotor")
)

type rotor struct {
	certManager certificate.Manager
}
