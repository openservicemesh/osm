package ads

// scLogger implements envoy control plane's log.Logger and delegates calls to the `log` variable defined in
// types.go. It is used for the envoy snapshot cache.
type scLogger struct{}

// Debugf logs a formatted debugging message.
func (l *scLogger) Debugf(format string, args ...interface{}) {
	log.Debug().Msgf(format, args...)
}

// Infof logs a formatted informational message.
func (l *scLogger) Infof(format string, args ...interface{}) {
	log.Info().Msgf(format, args...)
}

// Warnf logs a formatted warning message.
func (l *scLogger) Warnf(format string, args ...interface{}) {
	log.Warn().Msgf(format, args...)
}

// Errorf logs a formatted error message.
func (l *scLogger) Errorf(format string, args ...interface{}) {
	log.Error().Msgf(format, args...)
}
