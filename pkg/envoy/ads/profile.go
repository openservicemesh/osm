package ads

import (
	"fmt"
	"time"

	"github.com/openservicemesh/osm/pkg/metricsstore"
)

func xdsPathTimeTrack(t time.Time, tURIStr string, commonNameStr string, success *bool) {
	elapsed := time.Since(t)

	log.Debug().Msgf("[%s] proxy %s took %s",
		tURIStr,
		commonNameStr,
		elapsed)

	metricsstore.DefaultMetricsStore.ProxyConfigUpdateTime.
		WithLabelValues(commonNameStr, tURIStr, fmt.Sprintf("%t", *success)).
		Observe(elapsed.Seconds())
}
