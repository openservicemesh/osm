package metricsstore

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metricsRootNamespace is the root namespace for all the metrics emitted.
// Ex: osm_<metric-name>
const metricsRootNamespace = "osm"

// MetricsStore is a type that provides functionality related to metrics
type MetricsStore struct {
	k8sAPIEventCounter prometheus.Counter

	registry *prometheus.Registry
}

var defaultMetricsStore MetricsStore

// DefaultMetricsStore is the default metrics store
var DefaultMetricsStore = &defaultMetricsStore

func init() {
	defaultMetricsStore.k8sAPIEventCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsRootNamespace,
		Subsystem: "k8s",
		Name:      "api_event_count",
		Help:      "This counter represents the number of events received from the Kubernetes API Server",
	})

	defaultMetricsStore.registry = prometheus.NewRegistry()
}

// Start store
func (ms *MetricsStore) Start() {
	ms.registry.MustRegister(ms.k8sAPIEventCounter)
}

// Stop store
func (ms *MetricsStore) Stop() {
	ms.registry.Unregister(ms.k8sAPIEventCounter)
}

// IncK8sAPIEventCount increases the K8s API event counter
func (ms *MetricsStore) IncK8sAPIEventCount() {
	ms.k8sAPIEventCounter.Inc()
}

// Handler return the registry
func (ms *MetricsStore) Handler() http.Handler {
	return promhttp.InstrumentMetricHandler(
		ms.registry,
		promhttp.HandlerFor(ms.registry, promhttp.HandlerOpts{}),
	)
}
