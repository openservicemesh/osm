package k8s

import (
	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

func updateEventSpecificMetrics(eventType announcements.AnnouncementType) {
	switch eventType {
	case announcements.NamespaceAdded:
		metricsstore.DefaultMetricsStore.K8sMonitoredNamespaceCount.Inc()

	case announcements.NamespaceDeleted:
		metricsstore.DefaultMetricsStore.K8sMonitoredNamespaceCount.Dec()

	case announcements.PodAdded:
		metricsstore.DefaultMetricsStore.K8sMeshPodCount.Inc()

	case announcements.PodDeleted:
		metricsstore.DefaultMetricsStore.K8sMeshPodCount.Dec()

	case announcements.ServiceAdded:
		metricsstore.DefaultMetricsStore.K8sMeshServiceCount.Inc()

	case announcements.ServiceDeleted:
		metricsstore.DefaultMetricsStore.K8sMeshServiceCount.Dec()

	case announcements.ServiceAccountAdded:
		metricsstore.DefaultMetricsStore.K8sMeshServiceAccountCount.Inc()
	
	case announcements.ServiceAccountDeleted:
		metricsstore.DefaultMetricsStore.K8sMeshServiceAccountCount.Dec()

	case announcements.IngressAdded:
		metricsstore.DefaultMetricsStore.K8sMeshIngressCount.Inc()

	case announcements.IngressDeleted:
		metricsstore.DefaultMetricsStore.K8sMeshIngressCount.Dec()
	
	case announcements.EndpointAdded:
		metricsstore.DefaultMetricsStore.K8sMeshEndpointCount.Inc()

	case announcements.EndpointDeleted:
		metricsstore.DefaultMetricsStore.K8sMeshEndpointCount.Dec()
	}
}
