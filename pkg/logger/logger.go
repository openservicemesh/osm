package logger

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/open-service-mesh/osm/pkg/featureflags"
)

// CallerHook implements zerolog.Hook interface.
type CallerHook struct{}

// Run adds additional context
func (h CallerHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if _, file, line, ok := runtime.Caller(3); ok {
		e.Str("file", fmt.Sprintf("%s:%d", path.Base(file), line))
	}
}

// New creates a new zerolog.Logger
func New(component string) zerolog.Logger {
	l := log.With().Str("component", component).Logger().Hook(CallerHook{})
	if featureflags.EnableHumanReadalbeLog() {
		return l.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
	return l
}

// SetLogLevel sets the global logging level
func SetLogLevel(verbosity string) {
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
		log.Fatal().Msgf("Invalid log level '%s' specified. Please specify one of %v", verbosity, allowedLevels)
	}
}
