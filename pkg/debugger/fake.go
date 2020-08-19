package debugger

import (
	"net/http"
)

// FakeDebugServer is a fake debug server interface impl
type FakeDebugServer struct {
	Mappings map[string]http.Handler
}

// GetHandlers implements DebugServer interface and returns the inner object mappings
func (fds FakeDebugServer) GetHandlers() map[string]http.Handler {
	return fds.Mappings
}
