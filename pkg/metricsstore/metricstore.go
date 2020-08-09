package metricsstore

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/openservicemesh/osm/pkg/version"
)

// PrometheusNamespace is the Prometheus Namespace
var PrometheusNamespace = "osm"

// MetricStore is store maintaining all metrics
type MetricStore interface {
	Start()
	Stop()
	Handler() http.Handler
	SetUpdateLatencySec(time.Duration)
	IncK8sAPIEventCounter()
}

// OSMMetricsStore is store
type OSMMetricsStore struct {
	constLabels        prometheus.Labels
	updateLatency      prometheus.Gauge
	k8sAPIEventCounter prometheus.Counter

	registry *prometheus.Registry
}

// NewMetricStore returns a new metric store
func NewMetricStore(nameSpace string, podName string) MetricStore {
	constLabels := prometheus.Labels{
		"osm_namespace": nameSpace,
		"osm_pod":       podName,
		"osm_version":   fmt.Sprintf("%s/%s/%s", version.Version, version.GitCommit, version.BuildDate),
	}
	return &OSMMetricsStore{
		constLabels: constLabels,
		updateLatency: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   PrometheusNamespace,
			ConstLabels: constLabels,
			Name:        "update_latency_seconds",
			Help:        "The time spent in updating Envoy proxies",
		}),
		k8sAPIEventCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   PrometheusNamespace,
			ConstLabels: constLabels,
			Name:        "k8s_api_event_counter",
			Help:        "This counter represents the number of events received from Kubernetes API Server",
		}),
		registry: prometheus.NewRegistry(),
	}
}

// Start store
func (ms *OSMMetricsStore) Start() {
	ms.registry.MustRegister(ms.updateLatency)
	ms.registry.MustRegister(ms.k8sAPIEventCounter)
}

// Stop store
func (ms *OSMMetricsStore) Stop() {
	ms.registry.Unregister(ms.updateLatency)
	ms.registry.Unregister(ms.k8sAPIEventCounter)
}

// SetUpdateLatencySec updates latency
func (ms *OSMMetricsStore) SetUpdateLatencySec(duration time.Duration) {
	ms.updateLatency.Set(duration.Seconds())
}

// IncK8sAPIEventCounter increases the counter after receiving a k8s Event
func (ms *OSMMetricsStore) IncK8sAPIEventCounter() {
	ms.k8sAPIEventCounter.Inc()
}

// Handler return the registry
func (ms *OSMMetricsStore) Handler() http.Handler {
	return promhttp.InstrumentMetricHandler(
		ms.registry,
		promhttp.HandlerFor(ms.registry, promhttp.HandlerOpts{}),
	)
}
