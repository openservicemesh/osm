package generator

import (
	"fmt"
	"time"

	"github.com/jinzhu/copier"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

const (
	// MaxXdsLogsPerProxy keeps a higher bound of how many timestamps do we keep per proxy
	MaxXdsLogsPerProxy = 20
)

// GetXDSLog implements XDSDebugger interface and a log of the XDS responses sent to Envoy proxies.
func (g *EnvoyConfigGenerator) GetXDSLog() map[string]map[envoy.TypeURI][]time.Time {
	var logsCopy map[string]map[envoy.TypeURI][]time.Time
	var err error

	g.xdsMapLogMutex.Lock()
	// Making a copy to avoid debugger potential reads while writes are happening from XDS routines
	err = copier.Copy(&logsCopy, &g.xdsLog)
	g.xdsMapLogMutex.Unlock()

	if err != nil {
		log.Error().Err(err).Msgf("Failed to copy xdsLogMap")
	}

	return logsCopy
}

func (g *EnvoyConfigGenerator) trackXDSLog(proxyUUID string, typeURL envoy.TypeURI) {
	g.xdsMapLogMutex.Lock()
	defer g.xdsMapLogMutex.Unlock()
	if _, ok := g.xdsLog[proxyUUID]; !ok {
		g.xdsLog[proxyUUID] = make(map[envoy.TypeURI][]time.Time)
	}

	timeSlice, ok := g.xdsLog[proxyUUID][typeURL]
	if !ok {
		g.xdsLog[proxyUUID][typeURL] = []time.Time{time.Now()}
		return
	}

	timeSlice = append(timeSlice, time.Now())
	if len(timeSlice) > MaxXdsLogsPerProxy {
		timeSlice = timeSlice[1:]
	}
	g.xdsLog[proxyUUID][typeURL] = timeSlice
}

func xdsPathTimeTrack(startedAt time.Time, typeURI envoy.TypeURI, proxy *envoy.Proxy, success bool) {
	elapsed := time.Since(startedAt)

	log.Debug().Str("proxy", proxy.String()).Msgf("Time taken proxy to generate response for request with typeURI=%s: %s", typeURI, elapsed)

	metricsstore.DefaultMetricsStore.ProxyConfigUpdateTime.
		WithLabelValues(typeURI.String(), fmt.Sprintf("%t", success)).
		Observe(elapsed.Seconds())
}
