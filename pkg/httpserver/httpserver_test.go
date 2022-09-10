package httpserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/constants"
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

	metricsStore := metricsstore.DefaultMetricsStore
	metricsStore.Start(
		metricsstore.DefaultMetricsStore.HTTPResponseTotal,
		metricsstore.DefaultMetricsStore.HTTPResponseDuration,
	)

	httpServ := NewHTTPServer(testPort)

	httpServ.AddHandlers(map[string]http.Handler{
		"/alive": http.HandlerFunc(health.SimpleHandler),
	})

	httpServ.AddHandler(constants.MetricsPath, metricsStore.Handler())

	testServer := &httptest.Server{
		Config: httpServ.server,
	}

	//InvalidPath Check
	respL := recordCall(testServer, fmt.Sprintf("%s%s", url, invalidRoutePath))
	respHealth := recordCall(testServer, fmt.Sprintf("%s%s", url, "/alive"))

	assert.Equal(http.StatusNotFound, respL.StatusCode)
	assert.Equal(http.StatusOK, respHealth.StatusCode)
	// Ensure added metrics are generated based on previous requests in this
	// test
	assert.True(metricsStore.Contains(`osm_http_response_total{code="200",method="get",path="/alive"} 1` + "\n"))
	assert.True(metricsStore.Contains(`osm_http_response_duration_bucket{code="200",method="get",path="/alive",le="+Inf"} 1` + "\n"))

	err := httpServ.Start()
	assert.Nil(err)

	err = httpServ.Stop()
	assert.Nil(err)
	assert.False(httpServ.started)

	// test stopping a stopped server
	err = httpServ.Stop()
	assert.Nil(err)
}
