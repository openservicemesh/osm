package injector

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// Helper to parse timeout variable from webhook URL
func readTimeout(req *http.Request) (*time.Duration, error) {
	durationValue, found := req.URL.Query()[webhookTimeoutStr]
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
func webhookTimeTrack(start time.Time, timeout time.Duration) {
	elapsed := time.Since(start)
	var logEv *zerolog.Event
	percentOfTimeout := float64(elapsed.Microseconds()) / float64(timeout.Microseconds())

	if percentOfTimeout < 0.75 {
		logEv = log.Debug()
	} else if percentOfTimeout < 1 {
		logEv = log.Warn()
	} else {
		// Error logging when going beyond timeout value to process a webhook
		logEv = log.Error()
	}
	logEv.Msgf("Mutate Webhook took %v to execute (%.2f of it's timeout, %v)",
		elapsed, percentOfTimeout, timeout)
}
