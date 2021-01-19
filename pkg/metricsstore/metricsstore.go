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
	K8sAPIEventCounter *prometheus.CounterVec

	// K8sMonitoredNamespaceCount is the metric for the number of monitored namespaces
	K8sMonitoredNamespaceCount prometheus.Gauge

	// K8sMeshPodCount is the metric for the number of pods participating in the mesh
	K8sMeshPodCount prometheus.Gauge

	/*
	 * Proxy metrics
	 */
	// ProxyConnectCount is the metric for the total number of proxies connected to the controller
	ProxyConnectCount prometheus.Gauge

	// ProxyConfigUpdateTime is the histogram to track time spent for proxy configuration and its occurrences
	ProxyConfigUpdateTime *prometheus.HistogramVec

	/*
	 * Injector metrics
	 */
	// InjectorSidecarCount counts the number of injector webhooks dealt with over time
	InjectorSidecarCount prometheus.Counter

	// InjectorRqTime the histogram to track times for the injector webhook calls
	InjectorRqTime *prometheus.HistogramVec

	/*
	 * Certificate metrics
	 */
	// CertXdsIssuedCount is the metric counter for the number of xds certificates issued
	CertXdsIssuedCount prometheus.Counter

	// CertXdsIssuedCounter the histogram to track the time to issue xds certificates
	CertXdsIssuedTime *prometheus.HistogramVec

	/*
	 * MetricsStore internals should be defined below --------------
	 */
	registry *prometheus.Registry
}

var defaultMetricsStore MetricsStore

// DefaultMetricsStore is the default metrics store
var DefaultMetricsStore = &defaultMetricsStore

func init() {
	/*
	 * K8s metrics
	 */
	defaultMetricsStore.K8sAPIEventCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsRootNamespace,
			Subsystem: "k8s",
			Name:      "api_event_count",
			Help:      "represents the number of events received from the Kubernetes API Server",
		},
		[]string{"type", "namespace"},
	)
	defaultMetricsStore.K8sMonitoredNamespaceCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsRootNamespace,
		Subsystem: "k8s",
		Name:      "monitored_namespace_count",
		Help:      "represents the number of namespaces monitored by OSM controller",
	})
	defaultMetricsStore.K8sMeshPodCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsRootNamespace,
		Subsystem: "k8s",
		Name:      "mesh_pod_count",
		Help:      "represents the number of pods part of the mesh managed by OSM controller",
	})

	/*
	 * Proxy metrics
	 */
	defaultMetricsStore.ProxyConnectCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsRootNamespace,
		Subsystem: "proxy",
		Name:      "connect_count",
		Help:      "represents the number of proxies connected to OSM controller",
	})

	defaultMetricsStore.ProxyConfigUpdateTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsRootNamespace,
			Subsystem: "proxy",
			Name:      "config_update_time",
			Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 20, 40, 90},
			Help:      "Histogram to track time spent for proxy configuration",
		},
		[]string{
			"proxy_cn",      // proxy_cn is the common name of the proxy
			"resource_type", // identifies a typeURI resource
			"success",       // further labels if the operation succeeded or not
		})

	/*
	 * Injector metrics
	 */
	defaultMetricsStore.InjectorSidecarCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsRootNamespace,
		Subsystem: "injector",
		Name:      "injector_sidecar_count",
		Help:      "Counts the number of injector webhooks dealt with over time",
	})

	defaultMetricsStore.InjectorRqTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsRootNamespace,
			Subsystem: "injector",
			Name:      "injector_rq_time",
			Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 20, 40},
			Help:      "Histogram for time taken to perform sidecar injection",
		},
		[]string{
			"success",
		})

	/*
	 * Certificate metrics
	 */
	defaultMetricsStore.CertXdsIssuedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsRootNamespace,
		Subsystem: "cert",
		Name:      "xds_issued_count",
		Help:      "represents the total number of XDS certificates issued to proxies",
	})

	defaultMetricsStore.CertXdsIssuedTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsRootNamespace,
			Subsystem: "cert",
			Name:      "xds_issued_time",
			Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 20, 40, 90},
			Help:      "Histogram to track time spent to issue xds certificate",
		},
		[]string{
			"common_name", // common_name is the common name of the certificate
		})
	defaultMetricsStore.registry = prometheus.NewRegistry()
}

// Start store
func (ms *MetricsStore) Start() {
	ms.registry.MustRegister(ms.K8sAPIEventCounter)
	ms.registry.MustRegister(ms.K8sMonitoredNamespaceCount)
	ms.registry.MustRegister(ms.K8sMeshPodCount)
	ms.registry.MustRegister(ms.ProxyConnectCount)
	ms.registry.MustRegister(ms.ProxyConfigUpdateTime)
	ms.registry.MustRegister(ms.InjectorSidecarCount)
	ms.registry.MustRegister(ms.InjectorRqTime)
	ms.registry.MustRegister(ms.CertXdsIssuedCount)
	ms.registry.MustRegister(ms.CertXdsIssuedTime)
}

// Stop store
func (ms *MetricsStore) Stop() {
	ms.registry.Unregister(ms.K8sAPIEventCounter)
	ms.registry.Unregister(ms.K8sMonitoredNamespaceCount)
	ms.registry.Unregister(ms.K8sMeshPodCount)
	ms.registry.Unregister(ms.ProxyConnectCount)
	ms.registry.Unregister(ms.ProxyConfigUpdateTime)
	ms.registry.Unregister(ms.InjectorSidecarCount)
	ms.registry.Unregister(ms.InjectorRqTime)
	ms.registry.Unregister(ms.CertXdsIssuedCount)
	ms.registry.Unregister(ms.CertXdsIssuedTime)
}

// Handler return the registry
func (ms *MetricsStore) Handler() http.Handler {
	return promhttp.InstrumentMetricHandler(
		ms.registry,
		promhttp.HandlerFor(ms.registry, promhttp.HandlerOpts{}),
	)
}
