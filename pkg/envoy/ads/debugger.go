package ads

import (
	"time"

	"github.com/jinzhu/copier"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

// GetXDSLog implements XDSDebugger interface and a log of the XDS responses sent to Envoy proxies.
func (s *Server) GetXDSLog() *map[certificate.CommonName]map[envoy.TypeURI][]time.Time {
	var logsCopy map[certificate.CommonName]map[envoy.TypeURI][]time.Time
	var err error

	s.withXdsLogMutex(func() {
		// Making a copy to avoid debugger potential reads while writes are happening from XDS routines
		err = copier.Copy(&logsCopy, &s.xdsLog)
	})

	if err != nil {
		log.Err(err).Msgf("Failed to copy xdsLogMap")
	}

	return &logsCopy
}
