package ads

import (
	"github.com/rs/zerolog"
)

// scLogger implements envoy control plane's log.Logger and delegates calls to the `log` variable defined in
// types.go. It is used for the envoy snapshot cache.
type scLogger struct {
	log zerolog.Logger
}

// Debugf logs a formatted debugging message.
func (l *scLogger) Debugf(format string, args ...interface{}) {
	l.log.Debug().Msgf(format, args...)
}

// Infof logs a formatted informational message.
func (l *scLogger) Infof(format string, args ...interface{}) {
	l.log.Info().Msgf(format, args...)
}

// Warnf logs a formatted warning message.
func (l *scLogger) Warnf(format string, args ...interface{}) {
	l.log.Warn().Msgf(format, args...)
}

// Errorf logs a formatted error message.
func (l *scLogger) Errorf(format string, args ...interface{}) {
	l.log.Error().Msgf(format, args...)
}
