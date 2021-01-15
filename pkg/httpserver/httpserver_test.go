package httpserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/health"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

const (
	url              = "http://localhost"
	invalidRoutePath = "/debug/test2"
	testPort         = 9999

	// These paths are internal to debugServer
	readyPath   = "/health/ready"
	alivePath   = "/health/alive"
	metricsPath = "/metrics"
)

// Records an HTTP request and returns a response
func recordCall(ts *httptest.Server, path string) *http.Response {
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	ts.Config.Handler.ServeHTTP(w, req)

	return w.Result()
}

func TestNewHTTPServer(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	mockProbe := health.NewMockProbes(mockCtrl)
	testProbes := []health.Probes{mockProbe}
	metricsStore := metricsstore.DefaultMetricsStore

	httpServ := NewHTTPServer(testPort)

	httpServ.AddHandlers(map[string]http.Handler{
		"/health/ready": health.ReadinessHandler(testProbes, nil),
		"/health/alive": health.LivenessHandler(testProbes, nil),
		"/metrics":      metricsStore.Handler(),
	})

	testServer := &httptest.Server{
		Config: httpServ.server,
	}

	type newHTTPServerTest struct {
		readyLiveCheck     bool
		path               string
		expectedStatusCode int
	}

	//Readiness/Liveness Check
	newHTTPServerTests := []newHTTPServerTest{
		{true, readyPath, http.StatusOK},
		{false, readyPath, http.StatusServiceUnavailable},
		{true, alivePath, http.StatusOK},
		{false, alivePath, http.StatusServiceUnavailable},
	}

	for _, rt := range newHTTPServerTests {
		mockProbe.EXPECT().Readiness().Return(rt.readyLiveCheck).Times(1)
		mockProbe.EXPECT().Liveness().Return(rt.readyLiveCheck).Times(1)
		mockProbe.EXPECT().GetID().Return("test").Times(1)
		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, rt.path))

		assert.Equal(rt.expectedStatusCode, resp.StatusCode)
	}

	//InvalidPath Check
	mockProbe.EXPECT().Liveness().Times(0)
	mockProbe.EXPECT().Liveness().Times(0)
	mockProbe.EXPECT().GetID().Times(0)
	respL := recordCall(testServer, fmt.Sprintf("%s%s", url, invalidRoutePath))
	assert.Equal(http.StatusNotFound, respL.StatusCode)

	//Metrics path Check
	req := httptest.NewRequest("GET", fmt.Sprintf("%s%s", url, metricsPath), nil)
	w := httptest.NewRecorder()
	testServer.Config.Handler.ServeHTTP(w, req)
	respM := w.Result()
	assert.Equal(http.StatusOK, respM.StatusCode)
}
