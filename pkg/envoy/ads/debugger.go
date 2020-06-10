package ads

import (
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

// GetXDSLog implements XDSDebugger interface and a log of the XDS responses sent to Envoy proxies.
func (s Server) GetXDSLog() *map[certificate.CommonName]map[envoy.TypeURI][]time.Time {
	return &s.xdsLog
}
