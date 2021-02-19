package ads

import (
	"fmt"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

const (
	// MaxXdsLogsPerProxy keeps a higher bound of how many timestamps do we keep per proxy
	MaxXdsLogsPerProxy = 20
)

func xdsPathTimeTrack(t time.Time, tURIStr string, commonNameStr string, success *bool) {
	elapsed := time.Since(t)

	log.Debug().Msgf("[%s] proxy %s took %s",
		tURIStr,
		commonNameStr,
		elapsed)

	metricsstore.DefaultMetricsStore.ProxyConfigUpdateTime.
		WithLabelValues(tURIStr, fmt.Sprintf("%t", *success)).
		Observe(elapsed.Seconds())
}

func (s *Server) trackXDSLog(cn certificate.CommonName, typeURL envoy.TypeURI) {
	s.withXdsLogMutex(func() {
		if _, ok := s.xdsLog[cn]; !ok {
			s.xdsLog[cn] = make(map[envoy.TypeURI][]time.Time)
		}

		timeSlice, ok := s.xdsLog[cn][typeURL]
		if !ok {
			s.xdsLog[cn][typeURL] = []time.Time{time.Now()}
			return
		}

		timeSlice = append(timeSlice, time.Now())
		if len(timeSlice) > MaxXdsLogsPerProxy {
			timeSlice = timeSlice[1:]
		}
		s.xdsLog[cn][typeURL] = timeSlice
	})
}
