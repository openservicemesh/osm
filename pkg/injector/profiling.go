package injector

import (
	"fmt"
	"net/http"
	"time"

	"github.com/openservicemesh/osm/pkg/metricsstore"
)

var (
	defaultK8sTimeout = time.Duration(30 * time.Second)
)

// Helper to parse timeout variable from webhook URL
func readTimeout(req *http.Request) (time.Duration, error) {
	durationValue, found := req.URL.Query()[webhookMutateTimeoutKey]
	if !found || len(durationValue) != 1 {
		log.Error().Msg("Webhook timeout value not found in request")
		return defaultK8sTimeout, errParseWebhookTimeout
	}

	val, err := time.ParseDuration(durationValue[0])
	if err != nil {
		log.Error().Err(err).Msg("Error parsing timeout value as duration")
		return defaultK8sTimeout, errParseWebhookTimeout
	}
	return val, nil
}

// Time tracking function for webhook processing.
// Will calculate elapsed time since start and log debug how much time spent executing
func webhookTimeTrack(start time.Time, timeout time.Duration, success *bool) {
	elapsed := time.Since(start)
	percentOfTimeout := float64(elapsed.Microseconds()) / float64(timeout.Microseconds())

	log.Debug().Msgf("Mutate Webhook took %v to execute (%.2f of it's timeout, %v)",
		elapsed, percentOfTimeout, timeout)

	metricsstore.DefaultMetricsStore.InjectorSidecarCount.Inc()
	metricsstore.DefaultMetricsStore.InjectorRqTime.
		WithLabelValues(fmt.Sprintf("%t", *success)).Observe(elapsed.Seconds())
}
