package messaging

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

func updateNamespaceCounter(msg events.PubSubMessage) {
	switch msg.Kind {
	case announcements.NamespaceAdded:
		addedNamespaceObj, castOk := msg.NewObj.(*corev1.Namespace)
		if !castOk {
			log.Error().Msgf("Error casting to *corev1.Namespace added: got type %T", addedNamespaceObj)
		}
		namespace := addedNamespaceObj.GetName()
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter.WithLabelValues(namespace).Inc()
	case announcements.NamespaceDeleted:
		deletedNamespaceObj, castOk := msg.OldObj.(*corev1.Namespace)
		if !castOk {
			log.Error().Msgf("Error casting to *corev1.Namespace deleted: got type %T", deletedNamespaceObj)
		}
		namespace := deletedNamespaceObj.GetName()
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter.WithLabelValues(namespace).Dec()
	default:
		log.Error().Msgf("Unexpected PubSubMessage kind %s. Expected namespace-added or namespace-deleted", msg.Kind.String())
	}
}
