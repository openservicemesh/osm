package httpserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/health"
	"github.com/openservicemesh/osm/pkg/httpserver/constants"
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
	metricsStore.Start(
		metricsstore.DefaultMetricsStore.HTTPResponseTotal,
		metricsstore.DefaultMetricsStore.HTTPResponseDuration,
	)

	httpServ := NewHTTPServer(testPort)

	httpServ.AddHandlers(map[string]http.Handler{
		constants.HealthReadinessPath: health.ReadinessHandler(testProbes, nil),
		constants.HealthLivenessPath:  health.LivenessHandler(testProbes, nil),
	})

	httpServ.AddHandler(constants.MetricsPath, metricsStore.Handler())

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
		{true, constants.HealthReadinessPath, http.StatusOK},
		{false, constants.HealthReadinessPath, http.StatusServiceUnavailable},
		{true, constants.HealthLivenessPath, http.StatusOK},
		{false, constants.HealthLivenessPath, http.StatusServiceUnavailable},
	}

	for _, rt := range newHTTPServerTests {
		if rt.path == constants.HealthReadinessPath {
			mockProbe.EXPECT().Readiness().Return(rt.readyLiveCheck).Times(1)
		} else {
			mockProbe.EXPECT().Liveness().Return(rt.readyLiveCheck).Times(1)
		}
		if rt.expectedStatusCode == http.StatusServiceUnavailable {
			mockProbe.EXPECT().GetID().Return("test").Times(1)
		}
		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, rt.path))

		assert.Equal(rt.expectedStatusCode, resp.StatusCode)
	}

	//InvalidPath Check
	mockProbe.EXPECT().Liveness().Times(0)
	mockProbe.EXPECT().Liveness().Times(0)
	mockProbe.EXPECT().GetID().Times(0)
	respL := recordCall(testServer, fmt.Sprintf("%s%s", url, invalidRoutePath))
	assert.Equal(http.StatusNotFound, respL.StatusCode)

	// Ensure added metrics are generated based on previous requests in this
	// test
	assert.True(metricsStore.Contains(`osm_http_response_total{code="200",method="get",path="/health/alive"} 1` + "\n"))
	assert.True(metricsStore.Contains(`osm_http_response_duration_bucket{code="200",method="get",path="/health/alive",le="+Inf"} 1` + "\n"))

	err := httpServ.Start()
	assert.Nil(err)

	err = httpServ.Stop()
	assert.Nil(err)
	assert.False(httpServ.started)

	// test stopping a stopped server
	err = httpServ.Stop()
	assert.Nil(err)
}
