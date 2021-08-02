package metricsstore

import "github.com/prometheus/client_golang/prometheus"

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

	// K8sMeshServiceCount is the metric for the number of services in the mesh
	K8sMeshServiceCount prometheus.Gauge

	/*
	 * Proxy metrics
	 */
	// ProxyConnectCount is the metric for the total number of proxies connected to the controller
	ProxyConnectCount prometheus.Gauge

	// ProxyReconnectCount is the metric for the total reconnects from known proxies to the controller
	ProxyReconnectCount prometheus.Counter

	// ProxyConfigUpdateTime is the histogram to track time spent for proxy configuration and its occurrences
	ProxyConfigUpdateTime *prometheus.HistogramVec

	// ProxyBroadcastEventCounter is the metric for the total number of ProxyBroadcast events published
	ProxyBroadcastEventCount prometheus.Counter

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
	// CertIssuedCount is the metric counter for the number of certificates issued
	CertIssuedCount prometheus.Counter

	// CertXdsIssuedCounter the histogram to track the time to issue a certificates
	CertIssuedTime *prometheus.HistogramVec

	/*
	 * MetricsStore internals should be defined below --------------
	 */
	registry *prometheus.Registry
}
