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
	// Define metrics by their category below ----------------------

	/*
	 * K8s metrics
	 */
	// K8sAPIEventCounter is the metric counter for the number of K8s API events
	K8sAPIEventCounter prometheus.Counter

	// MetricsStore internals should be defined below --------
	registry *prometheus.Registry
}

var defaultMetricsStore MetricsStore

// DefaultMetricsStore is the default metrics store
var DefaultMetricsStore = &defaultMetricsStore

func init() {
	defaultMetricsStore.K8sAPIEventCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsRootNamespace,
		Subsystem: "k8s",
		Name:      "api_event_count",
		Help:      "This counter represents the number of events received from the Kubernetes API Server",
	})

	defaultMetricsStore.registry = prometheus.NewRegistry()
}

// Start store
func (ms *MetricsStore) Start() {
	ms.registry.MustRegister(ms.K8sAPIEventCounter)
}

// Stop store
func (ms *MetricsStore) Stop() {
	ms.registry.Unregister(ms.K8sAPIEventCounter)
}

// Handler return the registry
func (ms *MetricsStore) Handler() http.Handler {
	return promhttp.InstrumentMetricHandler(
		ms.registry,
		promhttp.HandlerFor(ms.registry, promhttp.HandlerOpts{}),
	)
}
