package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// WatchAndUpdateProxyBootstrapSecret watches for new pods being added to the mesh and updates
// their corresponding proxy bootstrap secret's OwnerReference to point to the associated pod.
func WatchAndUpdateProxyBootstrapSecret(kubeClient kubernetes.Interface, msgBroker *messaging.Broker, stop <-chan struct{}) {
	kubePubSub := msgBroker.GetKubeEventPubSub()
	podAddChan := kubePubSub.Sub(announcements.PodAdded.String())
	defer msgBroker.Unsub(kubePubSub, podAddChan)

	for {
		select {
		case <-stop:
			log.Info().Msg("Received stop signal, exiting proxy bootstrap secret update routine")
			return

		case podAddedMsg := <-podAddChan:
			psubMessage, castOk := podAddedMsg.(events.PubSubMessage)
			if !castOk {
				log.Error().Msgf("Error casting to events.PubSubMessage, got type %T", psubMessage)
				continue
			}

			// guaranteed can only be a PodAdded event
			addedPodObj, castOk := psubMessage.NewObj.(*corev1.Pod)
			if !castOk {
				log.Error().Msgf("Error casting to *corev1.Pod: got type %T", addedPodObj)
				continue
			}

			podUID := addedPodObj.GetUID()
			podUUID := addedPodObj.GetLabels()[constants.EnvoyUniqueIDLabelName]
			podName := addedPodObj.GetName()
			namespace := addedPodObj.GetNamespace()
			secretName := fmt.Sprintf("envoy-bootstrap-config-%s", podUUID)

			secret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
			if err != nil {
				log.Error().Err(err).Msgf("Failed to get secret %s/%s mounted to Pod %s/%s", namespace, secretName, namespace, podName)
				continue
			}

			secret.ObjectMeta.OwnerReferences = append(secret.ObjectMeta.OwnerReferences, metav1.OwnerReference{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       podName,
				UID:        podUID,
			})

			if _, err = kubeClient.CoreV1().Secrets(namespace).Update(context.Background(), secret, metav1.UpdateOptions{}); err != nil {
				// There might be conflicts when multiple controllers try to update the same resource
				// One of the controllers will successfully update the resource, hence conflicts shoud be ignored and not treated as an error
				if !apierrors.IsConflict(err) {
					log.Error().Err(err).Msgf("Failed to update OwnerReference for Secret %s/%s to reference Pod %s/%s", namespace, secretName, namespace, podName)
				}
			} else {
				log.Debug().Msgf("Updated OwnerReference for Secret %s/%s to reference Pod %s/%s", namespace, secretName, namespace, podName)
			}
		}
	}
}

// WatchAndUpdateLogLevel watches for log level changes and updates the global log level
func WatchAndUpdateLogLevel(msgBroker *messaging.Broker, stop <-chan struct{}) {
	kubePubSub := msgBroker.GetKubeEventPubSub()
	meshCfgUpdateChan := kubePubSub.Sub(announcements.MeshConfigUpdated.String())
	defer msgBroker.Unsub(kubePubSub, meshCfgUpdateChan)

	for {
		select {
		case <-stop:
			log.Info().Msg("Received stop signal, exiting log level update routine")
			return

		case event := <-meshCfgUpdateChan:
			msg, ok := event.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Error casting to PubSubMessage, got type %T", msg)
				continue
			}

			prevObj, prevOk := msg.OldObj.(*configv1alpha3.MeshConfig)
			newObj, newOk := msg.NewObj.(*configv1alpha3.MeshConfig)
			if !prevOk || !newOk {
				log.Error().Msgf("Error casting to *MeshConfig, got type prev=%T, new=%T", prevObj, newObj)
			}

			// Update the log level if necessary
			if prevObj.Spec.Observability.OSMLogLevel != newObj.Spec.Observability.OSMLogLevel {
				if err := logger.SetLogLevel(newObj.Spec.Observability.OSMLogLevel); err != nil {
					log.Error().Err(err).Msgf("Error setting controller log level to %s", newObj.Spec.Observability.OSMLogLevel)
				}
			}
		}
	}
}
