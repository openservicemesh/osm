package k8s

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
)

var _ = Describe("Test Announcement Handlers", func() {
	const envoyBootstrapConfigVolume = "envoy-bootstrap-config-volume"

	var kubeClient kubernetes.Interface
	var stopChannel chan struct{}

	Context("test patchSecret()", func() {
		BeforeEach(func() {
			kubeClient = fake.NewSimpleClientset()
			stopChannel = PatchSecretHandler(kubeClient)
		})

		AfterEach(func() {
			stopChannel <- struct{}{}
		})

		It("verifies the secrets have been patched with OwnerReference", func() {
			podName := "app"
			namespace := "app"
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
			Expect(err).To(BeNil())
			_, err = kubeClient.CoreV1().Pods(namespace).Create(context.Background(), &pod, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			// Publish a podAdded event
			events.Publish(events.PubSubMessage{
				AnnouncementType: announcements.PodAdded,
				NewObj:           &pod,
				OldObj:           nil,
			})

			expectedOwnerReference := metav1.OwnerReference{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       podName,
				UID:        pod.UID,
			}

			// Expect the OwnerReference to be updated eventually
			Eventually(func() bool {
				secret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, ownerReference := range secret.GetOwnerReferences() {
					if reflect.DeepEqual(ownerReference, expectedOwnerReference) {
						return true
					}
				}
				return false
			}).Should(BeTrue())
		})
	})
})
