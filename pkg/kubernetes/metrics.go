package kubernetes

import (
	"github.com/openservicemesh/osm/pkg/dispatcher"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

func updateEventSpecificMetrics(eventType dispatcher.AnnouncementType) {
	switch eventType {
	case dispatcher.NamespaceAdded:
		metricsstore.DefaultMetricsStore.K8sMonitoredNamespaceCount.Inc()

	case dispatcher.NamespaceDeleted:
		metricsstore.DefaultMetricsStore.K8sMonitoredNamespaceCount.Dec()

	case dispatcher.PodAdded:
		metricsstore.DefaultMetricsStore.K8sMeshPodCount.Inc()

	case dispatcher.PodDeleted:
		metricsstore.DefaultMetricsStore.K8sMeshPodCount.Dec()
	}
}
