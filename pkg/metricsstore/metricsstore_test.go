package metricsstore

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() {
	DefaultMetricsStore.Start()
}

func teardown() {
	DefaultMetricsStore.Stop()
}

func TestK8sAPIEventCounter(t *testing.T) {
	assert := tassert.New(t)

	apiEventCount := 3

	for i := 1; i <= apiEventCount; i++ {
		DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues("add", "foo").Inc()

		handler := DefaultMetricsStore.Handler()

		req, err := http.NewRequest("GET", "/metrics", nil)
		assert.Nil(err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(http.StatusOK, rr.Code)

		expectedResp := fmt.Sprintf(`# HELP osm_k8s_api_event_count represents the number of events received from the Kubernetes API Server
# TYPE osm_k8s_api_event_count counter
osm_k8s_api_event_count{namespace="foo",type="add"} %d
`, i /* api event count */)
		assert.Contains(rr.Body.String(), expectedResp)
	}
}

func TestProxyConnectCount(t *testing.T) {
	assert := tassert.New(t)

	proxiesConnected := 5
	proxiesDisconnected := 2

	for i := 1; i <= proxiesConnected; i++ {
		DefaultMetricsStore.ProxyConnectCount.Inc()
	}
	for i := 1; i <= proxiesDisconnected; i++ {
		DefaultMetricsStore.ProxyConnectCount.Dec()
	}

	handler := DefaultMetricsStore.Handler()

	req, err := http.NewRequest("GET", "/metrics", nil)
	assert.Nil(err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(http.StatusOK, rr.Code)

	expectedResp := fmt.Sprintf(`# HELP osm_proxy_connect_count represents the number of proxies connected to OSM controller
# TYPE osm_proxy_connect_count gauge
osm_proxy_connect_count %d
`, proxiesConnected-proxiesDisconnected)
	assert.Contains(rr.Body.String(), expectedResp)
}
