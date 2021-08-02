// Package metricsstore implements a Prometheus metrics store for OSM's control plane metrics.
package metricsstore

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metricsRootNamespace is the root namespace for all the metrics emitted.
// Ex: osm_<metric-name>
const metricsRootNamespace = "osm"

var metricsStore *MetricsStore

// GetMetricsStore returns a singleton
func GetMetricsStore() *MetricsStore {
	if metricsStore == nil {
		metricsStore = &MetricsStore{
			/*
			 * K8s metrics
			 */
			K8sAPIEventCounter: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: metricsRootNamespace,
					Subsystem: "k8s",
					Name:      "api_event_count",
					Help:      "Number of events received from the Kubernetes API Server",
				},
				[]string{"type", "namespace"},
			),

			K8sMonitoredNamespaceCount: prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: metricsRootNamespace,
				Subsystem: "k8s",
				Name:      "monitored_namespace_count",
				Help:      "Number of namespaces monitored by OSM controller",
			}),

			K8sMeshPodCount: prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: metricsRootNamespace,
				Subsystem: "k8s",
				Name:      "mesh_pod_count",
				Help:      "Number of pods part of the mesh managed by OSM controller",
			}),

			K8sMeshServiceCount: prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: metricsRootNamespace,
				Subsystem: "k8s",
				Name:      "mesh_service_count",
				Help:      "Number of services part of the mesh managed by OSM controller",
			}),

			/*
			 * Proxy metrics
			 */
			ProxyConnectCount: prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: metricsRootNamespace,
				Subsystem: "proxy",
				Name:      "connect_count",
				Help:      "Number of proxies connected to OSM controller",
			}),

			ProxyReconnectCount: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: metricsRootNamespace,
				Subsystem: "proxy",
				Name:      "reconnect_count",
				Help:      "Number of reconnects from known proxies to OSM controller",
			}),

			ProxyConfigUpdateTime: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: metricsRootNamespace,
					Subsystem: "proxy",
					Name:      "config_update_time",
					Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 20, 40, 90},
					Help:      "Histogram for time spent for proxy configuration",
				},
				[]string{
					"resource_type", // identifies a typeURI resource
					"success",       // further labels if the operation succeeded or not
				}),

			ProxyBroadcastEventCount: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: metricsRootNamespace,
				Subsystem: "proxy",
				Name:      "broadcast_event_count",
				Help:      "Number of ProxyBroadcast events published by the OSM controller",
			}),

			/*
			 * Injector metrics
			 */
			InjectorSidecarCount: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: metricsRootNamespace,
				Subsystem: "injector",
				Name:      "injector_sidecar_count",
				Help:      "Counts the number of injector webhooks dealt with over time",
			}),

			InjectorRqTime: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: metricsRootNamespace,
					Subsystem: "injector",
					Name:      "injector_rq_time",
					Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 20, 40},
					Help:      "Histogram for time taken to perform sidecar injection",
				},
				[]string{
					"success",
				}),

			/*
			 * Certificate metrics
			 */
			CertIssuedCount: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: metricsRootNamespace,
				Subsystem: "cert",
				Name:      "issued_count",
				Help:      "Total number of XDS certificates issued to proxies",
			}),

			CertIssuedTime: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: metricsRootNamespace,
					Subsystem: "cert",
					Name:      "issued_time",
					Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 20, 40, 90},
					Help:      "Histogram for time spent to issue xds certificate",
				},
				[]string{}),

			registry: prometheus.NewRegistry(),
		}
	}
	return metricsStore
}

// Start store
func (ms *MetricsStore) Start(cs ...prometheus.Collector) {
	ms.registry.MustRegister(cs...)
}

// Stop store
func (ms *MetricsStore) Stop(cs ...prometheus.Collector) {
	for _, c := range cs {
		ms.registry.Unregister(c)
	}
}

// Handler return the registry
func (ms *MetricsStore) Handler() http.Handler {
	return promhttp.InstrumentMetricHandler(
		ms.registry,
		promhttp.HandlerFor(ms.registry, promhttp.HandlerOpts{}),
	)
}
