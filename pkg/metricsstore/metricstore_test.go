package metricsstore

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("", func() {
	Context("", func() {
		It("", func() {
			metricsStore := NewMetricStore("a", "b")
			metricsStore.Start()
			metricsStore.SetUpdateLatencySec(1 * time.Second)

			handler := metricsStore.Handler()

			req, err := http.NewRequest("GET", "/metrics", nil)
			Expect(err).ToNot(HaveOccurred())

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			expected := `# HELP osm_k8s_api_event_counter This counter represents the number of events received from Kubernetes API Server
# TYPE osm_k8s_api_event_counter counter
osm_k8s_api_event_counter{osm_namespace="a",osm_pod="b",osm_version="//"} 0
# HELP osm_update_latency_seconds The time spent in updating Envoy proxies
# TYPE osm_update_latency_seconds gauge
osm_update_latency_seconds{osm_namespace="a",osm_pod="b",osm_version="//"} 1
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 0
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
`

			Expect(rr.Body.String()).To(Equal(expected), fmt.Sprintf("Actual: %s", rr.Body.String()))

			metricsStore.Stop()
		})
	})
})
