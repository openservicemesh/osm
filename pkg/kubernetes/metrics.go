package kubernetes

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
	}
}
