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
		HealthReadinessPath: health.ReadinessHandler(testProbes, nil),
		HealthLivenessPath:  health.LivenessHandler(testProbes, nil),
	})

	httpServ.AddHandler(MetricsPath, metricsStore.Handler())

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
		{true, HealthReadinessPath, http.StatusOK},
		{false, HealthReadinessPath, http.StatusServiceUnavailable},
		{true, HealthLivenessPath, http.StatusOK},
		{false, HealthLivenessPath, http.StatusServiceUnavailable},
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
	req := httptest.NewRequest("GET", fmt.Sprintf("%s%s", url, MetricsPath), nil)
	w := httptest.NewRecorder()
	testServer.Config.Handler.ServeHTTP(w, req)
	respM := w.Result()
	assert.Equal(http.StatusOK, respM.StatusCode)

	err := httpServ.Start()
	assert.Nil(err)

	err = httpServ.Stop()
	assert.Nil(err)
	assert.False(httpServ.started)

	// test stopping a stopped server
	err = httpServ.Stop()
	assert.Nil(err)
}
