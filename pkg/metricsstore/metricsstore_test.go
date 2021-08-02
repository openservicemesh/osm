package metricsstore

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func setup() {
	GetMetricsStore().Start(
		GetMetricsStore().K8sAPIEventCounter,
		GetMetricsStore().ProxyConnectCount,
	)
}

func teardown() {
	GetMetricsStore().Stop(
		GetMetricsStore().K8sAPIEventCounter,
		GetMetricsStore().ProxyConnectCount,
	)
}

func TestMetricsStore(t *testing.T) {
	setup()
	defer teardown()

	t.Run("K8sAPIEventCounter", func(t *testing.T) {
		assert := tassert.New(t)

		apiEventCount := 3

		for i := 1; i <= apiEventCount; i++ {
			GetMetricsStore().K8sAPIEventCounter.WithLabelValues("add", "foo").Inc()

			handler := GetMetricsStore().Handler()

			req, err := http.NewRequest("GET", "/metrics", nil)
			assert.Nil(err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(http.StatusOK, rr.Code)

			expectedResp := fmt.Sprintf(`# HELP osm_k8s_api_event_count Represents the number of events received from the Kubernetes API Server
# TYPE osm_k8s_api_event_count counter
osm_k8s_api_event_count{namespace="foo",type="add"} %d
`, i /* api event count */)
			assert.Contains(rr.Body.String(), expectedResp)
		}
	})

	t.Run("ProxyConnectCount", func(t *testing.T) {
		assert := tassert.New(t)

		proxiesConnected := 5
		proxiesDisconnected := 2

		for i := 1; i <= proxiesConnected; i++ {
			GetMetricsStore().ProxyConnectCount.Inc()
		}
		for i := 1; i <= proxiesDisconnected; i++ {
			GetMetricsStore().ProxyConnectCount.Dec()
		}

		handler := GetMetricsStore().Handler()

		req, err := http.NewRequest("GET", "/metrics", nil)
		assert.Nil(err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(http.StatusOK, rr.Code)

		expectedResp := fmt.Sprintf(`# HELP osm_proxy_connect_count Represents the number of proxies connected to OSM controller
# TYPE osm_proxy_connect_count gauge
osm_proxy_connect_count %d
`, proxiesConnected-proxiesDisconnected)
		assert.Contains(rr.Body.String(), expectedResp)
	})
}
