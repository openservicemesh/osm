package resourcemetrics

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// StartNamespaceCounter updates the MonitoredNamespaceCounter based on the namespace added and
// deleted k8s events
func StartNamespaceCounter(msgBroker *messaging.Broker, stop <-chan struct{}) {
	kubePubSub := msgBroker.GetKubeEventPubSub()
	namespaceEventChan := kubePubSub.Sub(
		announcements.NamespaceAdded.String(),
		announcements.NamespaceDeleted.String())
	defer msgBroker.Unsub(kubePubSub, namespaceEventChan)

	for {
		select {
		case <-stop:
			return

		case namespaceEvent := <-namespaceEventChan:
			psubMessage, castOk := namespaceEvent.(events.PubSubMessage)
			if !castOk {
				log.Error().Msgf("Error casting to events.PubSubMessage, got type %T", psubMessage)
				continue
			}

			switch psubMessage.Kind {
			case announcements.NamespaceAdded:
				addedNamespaceObj, castOk := psubMessage.NewObj.(*corev1.Namespace)
				if !castOk {
					log.Error().Msgf("Error casting to *corev1.Namespace added: got type %T", addedNamespaceObj)
					continue
				}
				namespace := addedNamespaceObj.GetName()
				metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter.WithLabelValues(namespace).Inc()
			case announcements.NamespaceDeleted:
				deletedNamespaceObj, castOk := psubMessage.OldObj.(*corev1.Namespace)
				if !castOk {
					log.Error().Msgf("Error casting to *corev1.Namespace deleted: got type %T", deletedNamespaceObj)
					continue
				}
				namespace := deletedNamespaceObj.GetName()
				metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter.WithLabelValues(namespace).Dec()
			default:
				log.Error().Msgf("Unexpected PubSubMessage kind %s. Expected namespace-added or namespace-deleted", psubMessage.Kind.String())
			}
		}
	}
}
