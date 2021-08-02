package k8s

import (
	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

func updateEventSpecificMetrics(eventType announcements.AnnouncementType) {
	switch eventType {
	case announcements.NamespaceAdded:
		metricsstore.GetMetricsStore().K8sMonitoredNamespaceCount.Inc()

	case announcements.NamespaceDeleted:
		metricsstore.GetMetricsStore().K8sMonitoredNamespaceCount.Dec()

	case announcements.PodAdded:
		metricsstore.GetMetricsStore().K8sMeshPodCount.Inc()

	case announcements.PodDeleted:
		metricsstore.GetMetricsStore().K8sMeshPodCount.Dec()

	case announcements.ServiceAdded:
		metricsstore.GetMetricsStore().K8sMeshServiceCount.Inc()

	case announcements.ServiceDeleted:
		metricsstore.GetMetricsStore().K8sMeshServiceCount.Dec()
	}
}
