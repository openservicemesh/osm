package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

// PatchSecretHandler patches the envoy bootstrap config secrets based on the PodAdd events
func PatchSecretHandler(kubeClient kubernetes.Interface) {
	podAddSubscription := events.GetPubSubInstance().Subscribe(announcements.PodAdded)

	go func() {
		for {
			podAddedMsg := <-podAddSubscription
			psubMessage, castOk := podAddedMsg.(events.PubSubMessage)
			if !castOk {
				log.Error().Msgf("Error casting PubSubMessage: %T %v", psubMessage, psubMessage)
				continue
			}

			// guaranteed can only be a PodAdded event
			addedPodObj, castOk := psubMessage.NewObj.(*corev1.Pod)
			if !castOk {
				log.Error().Msgf("Failed to cast to *v1.Pod: %T %v", psubMessage.OldObj, psubMessage.OldObj)
				continue
			}

			podUID := addedPodObj.GetUID()
			podUUID := addedPodObj.GetLabels()[constants.EnvoyUniqueIDLabelName]
			podName := addedPodObj.GetName()
			namespace := addedPodObj.GetNamespace()
			secretName := fmt.Sprintf("envoy-bootstrap-config-%s", podUUID)

			if secret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{}); err != nil {
				log.Error().Err(err).Msgf("Failed to get secret %s/%s mounted to Pod %s/%s", namespace, secretName, namespace, podName)
			} else {
				secret.ObjectMeta.OwnerReferences = append(secret.ObjectMeta.OwnerReferences, metav1.OwnerReference{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       podName,
					UID:        podUID,
				})

				if _, err = kubeClient.CoreV1().Secrets(namespace).Update(context.Background(), secret, metav1.UpdateOptions{}); err != nil {
					log.Error().Err(err).Msgf("Failed to update OwnerReference for Secret %s/%s to reference Pod %s/%s", namespace, secretName, namespace, podName)
				} else {
					log.Debug().Msgf("Updated OwnerReference for Secret %s/%s to reference Pod %s/%s", namespace, secretName, namespace, podName)
				}
			}
		}
	}()
}
