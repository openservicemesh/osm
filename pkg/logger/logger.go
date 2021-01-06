package logger

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/openservicemesh/osm/pkg/constants"
)

// CallerHook implements zerolog.Hook interface.
type CallerHook struct{}

// Run adds additional context
func (h CallerHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	if _, file, line, ok := runtime.Caller(3); ok {
		e.Str("file", fmt.Sprintf("%s:%d", path.Base(file), line))
	}
}

func newLogger(component string) zerolog.Logger {
	return log.With().Str("component", component).Logger().Hook(CallerHook{})
}

// New creates a new zerolog.Logger
func New(component string) zerolog.Logger {
	if os.Getenv(constants.EnvVarHumanReadableLogMessages) == "true" {
		return NewPretty(component)
	}
	return newLogger(component)
}

// NewPretty creates a new zerolog.Logger, which emits human-readable log messages
func NewPretty(component string) zerolog.Logger {
	return newLogger(component).Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

// SetLogLevel sets the global logging level
func SetLogLevel(verbosity string) error {
	switch strings.ToLower(verbosity) {
	// DebugLevel defines debug log level.
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)

	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)

	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)

	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)

	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)

	case "disabled":
		zerolog.SetGlobalLevel(zerolog.Disabled)

	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)

	default:
		allowedLevels := []string{"debug", "info", "warn", "error", "fatal", "panic", "disabled", "trace"}
		return errors.Errorf("Invalid log level '%s' specified. Please specify one of %v", verbosity, allowedLevels)
	}
	return nil
}
