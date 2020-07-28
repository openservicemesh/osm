package ads

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

// GetXDSLog implements XDSDebugger interface and a log of the XDS responses sent to Envoy proxies.
func (s Server) GetXDSLog() *map[certificate.CommonName]map[envoy.TypeURI][]time.Time {
	return &s.xdsLog
}
