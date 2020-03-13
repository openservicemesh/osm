package metricsstore

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/open-service-mesh/osm/pkg/version"
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

// SMCMetricsStore is store
type SMCMetricsStore struct {
	constLabels        prometheus.Labels
	updateLatency      prometheus.Gauge
	k8sAPIEventCounter prometheus.Counter

	registry *prometheus.Registry
}

// NewMetricStore returns a new metric store
func NewMetricStore(nameSpace string, podName string) MetricStore {
	constLabels := prometheus.Labels{
		"smc_namespace": nameSpace,
		"smc_pod":       podName,
		"smc_version":   fmt.Sprintf("%s/%s/%s", version.Version, version.GitCommit, version.BuildDate),
	}
	return &SMCMetricsStore{
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
func (ms *SMCMetricsStore) Start() {
	ms.registry.MustRegister(ms.updateLatency)
	ms.registry.MustRegister(ms.k8sAPIEventCounter)
}

// Stop store
func (ms *SMCMetricsStore) Stop() {
	ms.registry.Unregister(ms.updateLatency)
	ms.registry.Unregister(ms.k8sAPIEventCounter)
}

// SetUpdateLatencySec updates latency
func (ms *SMCMetricsStore) SetUpdateLatencySec(duration time.Duration) {
	ms.updateLatency.Set(duration.Seconds())
}

// IncK8sAPIEventCounter increases the counter after recieving a k8s Event
func (ms *SMCMetricsStore) IncK8sAPIEventCounter() {
	ms.k8sAPIEventCounter.Inc()
}

// Handler return the registry
func (ms *SMCMetricsStore) Handler() http.Handler {
	return promhttp.InstrumentMetricHandler(
		ms.registry,
		promhttp.HandlerFor(ms.registry, promhttp.HandlerOpts{}),
	)
}
