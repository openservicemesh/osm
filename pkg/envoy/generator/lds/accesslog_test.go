package lds

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessLogBuild(t *testing.T) {
	testCases := []struct {
		name                  string
		ab                    *accessLogBuilder
		numExpectedAccessLogs int
		expectErr             bool
	}{
		{
			name:                  "default stream access log and format",
			ab:                    NewAccessLogBuilder().Name("foo"),
			numExpectedAccessLogs: 1, // STDOUT stream access log
			expectErr:             false,
		},
		{
			name:                  "custom stream access log text format",
			ab:                    NewAccessLogBuilder().Name("foo").Format("[%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS%\n"),
			numExpectedAccessLogs: 1, // STDOUT stream access log
			expectErr:             false,
		},
		{
			name:                  "custom  stream access log JSON format",
			ab:                    NewAccessLogBuilder().Name("foo").Format(`{"authority":"%REQ(:AUTHORITY)%","bytes_received":"%BYTES_RECEIVED%","bytes_sent":"%BYTES_SENT%"}`),
			numExpectedAccessLogs: 1, // STDOUT stream access log
			expectErr:             false,
		},
		{
			name:                  "default stream access log and OpenTelemetry access log",
			ab:                    NewAccessLogBuilder().Name("foo").OpenTelemetryCluster("otel-collector"),
			numExpectedAccessLogs: 2, // STDOUT stream access log and OpenTelemetry access log
			expectErr:             false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			accessLogs, err := tc.ab.Build()

			a.Len(accessLogs, tc.numExpectedAccessLogs)
			a.Equal(tc.expectErr, err != nil)
		})
	}
}
