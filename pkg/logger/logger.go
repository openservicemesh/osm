package logger

import (
	"os"
	"path"
	"runtime"

	"github.com/rs/zerolog"
)

// CallerHook implements zerolog.Hook interface.
type CallerHook struct{}

// Run adds additional context
func (h CallerHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if _, file, line, ok := runtime.Caller(3); ok {
		e.Str("file", path.Base(file)).Int("line", line)
	}
}

// New creates a new zerolog.Logger
func New(name string) zerolog.Logger {
	l := zerolog.New(os.Stdout).With().Timestamp().Str("logger", name).Logger().Hook(CallerHook{})
	return l.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}
