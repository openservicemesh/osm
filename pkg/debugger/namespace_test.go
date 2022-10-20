package debugger

import (
	"net/http/httptest"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

// Tests getMonitoredNamespaces through HTTP handler returns a the list of monitored namespaces
func TestMonitoredNamespaceHandler(t *testing.T) {
	assert := tassert.New(t)

	ds := DebugConfig{}
	monitoredNamespacesHandler := ds.getMonitoredNamespacesHandler()

	responseRecorder := httptest.NewRecorder()
	monitoredNamespacesHandler.ServeHTTP(responseRecorder, nil)
	actualResponseBody := responseRecorder.Body.String()
	expectedResponseBody := `{"namespaces":["default"]}`
	assert.Equal(expectedResponseBody, actualResponseBody, "Actual value did not match expectations:\n%s", actualResponseBody)
}
