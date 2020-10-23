package injector

import (
	"net/http"
	"time"
)

// Helper to parse timeout variable from webhook URL
func readTimeout(req *http.Request) (*time.Duration, error) {
	durationValue, found := req.URL.Query()[webhookMutateTimeoutKey]
	if !found || len(durationValue) != 1 {
		log.Error().Msgf("Webhook timeout value not found in request")
		return nil, errParseWebhookTimeout
	}

	val, err := time.ParseDuration(durationValue[0])
	if err != nil {
		log.Error().Msgf("Error parsing timeout value as duration: %v", err)
		return nil, errParseWebhookTimeout
	}
	return &val, nil
}

// Time tracking function for webhook processing.
// Will calculate elapsed time since start and log debug how much time spent executing
func webhookTimeTrack(start time.Time, timeout time.Duration) {
	elapsed := time.Since(start)
	percentOfTimeout := float64(elapsed.Microseconds()) / float64(timeout.Microseconds())

	log.Debug().Msgf("Mutate Webhook took %v to execute (%.2f of it's timeout, %v)",
		elapsed, percentOfTimeout, timeout)
}
