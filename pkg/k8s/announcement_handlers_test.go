package k8s

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

func TestWatchAndUpdateProxyBootstrapSecret(t *testing.T) {
	a := assert.New(t)

	stop := make(chan struct{})
	defer close(stop)

	msgBroker := messaging.NewBroker(stop)
	kubeClient := fake.NewSimpleClientset()

	// Start the function being tested
	go WatchAndUpdateProxyBootstrapSecret(kubeClient, msgBroker, stop)
	// Subscription should happen before an event is published by the test, so
	// add a delay before the test triggers events
	time.Sleep(500 * time.Millisecond)

	podName := "app"
	namespace := "app"
	envoyBootstrapConfigVolume := "envoy-bootstrap-config-volume"
	podUUID := uuid.New().String()
	podUID := uuid.New().String()
	secretName := fmt.Sprintf("envoy-bootstrap-config-%s", podUUID)

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels:    map[string]string{constants.EnvoyUniqueIDLabelName: podUUID},
			UID:       types.UID(podUID),
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: envoyBootstrapConfigVolume,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secretName,
						},
					},
				},
			},
		},
	}

	_, err := kubeClient.CoreV1().Secrets(namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
	a.Nil(err)
	_, err = kubeClient.CoreV1().Pods(namespace).Create(context.Background(), &pod, metav1.CreateOptions{})
	a.Nil(err)

	// Publish a podAdded event
	msgBroker.GetKubeEventPubSub().Pub(events.PubSubMessage{
		Kind:   announcements.PodAdded,
		NewObj: &pod,
		OldObj: nil,
	}, announcements.PodAdded.String())

	expectedOwnerReference := metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Name:       podName,
		UID:        pod.UID,
	}

	// Expect the OwnerReference to be updated eventually
	a.Eventually(func() bool {
		secret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
		a.Nil(err)
		for _, ownerReference := range secret.GetOwnerReferences() {
			if reflect.DeepEqual(ownerReference, expectedOwnerReference) {
				return true
			}
		}
		return false
	}, 1*time.Second, 50*time.Millisecond)
}

func TestWatchAndUpdateLogLevel(t *testing.T) {
	testCases := []struct {
		name             string
		event            events.PubSubMessage
		expectedLogLevel zerolog.Level
	}{
		{
			name: "log level updated to trace",
			event: events.PubSubMessage{
				Kind: announcements.MeshConfigUpdated,
				OldObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						Observability: configv1alpha3.ObservabilitySpec{
							OSMLogLevel: "info",
						},
					},
				},
				NewObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						Observability: configv1alpha3.ObservabilitySpec{
							OSMLogLevel: "trace",
						},
					},
				},
			},
			expectedLogLevel: zerolog.TraceLevel,
		},
		{
			name: "log level updated to info",
			event: events.PubSubMessage{
				Kind: announcements.MeshConfigUpdated,
				OldObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						Observability: configv1alpha3.ObservabilitySpec{
							OSMLogLevel: "trace",
						},
					},
				},
				NewObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						Observability: configv1alpha3.ObservabilitySpec{
							OSMLogLevel: "info",
						},
					},
				},
			},
			expectedLogLevel: zerolog.InfoLevel,
		},
		{
			name: "log level unchanged",
			event: events.PubSubMessage{
				Kind: announcements.MeshConfigUpdated,
				OldObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						Observability: configv1alpha3.ObservabilitySpec{
							OSMLogLevel: "info",
						},
					},
				},
				NewObj: &configv1alpha3.MeshConfig{
					Spec: configv1alpha3.MeshConfigSpec{
						Observability: configv1alpha3.ObservabilitySpec{
							OSMLogLevel: "info",
						},
					},
				},
			},
			expectedLogLevel: zerolog.InfoLevel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			stop := make(chan struct{})
			defer close(stop)
			msgBroker := messaging.NewBroker(stop)

			go WatchAndUpdateLogLevel(msgBroker, stop)
			// Subscription should happen before an event is published by the test, so
			// add a delay before the test triggers events
			time.Sleep(500 * time.Millisecond)

			msgBroker.GetKubeEventPubSub().Pub(tc.event, announcements.MeshConfigUpdated.String())

			a.Eventually(func() bool {
				return zerolog.GlobalLevel() == tc.expectedLogLevel
			}, 1*time.Second, 50*time.Millisecond)
		})
	}
}
