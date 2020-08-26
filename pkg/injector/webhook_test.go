package injector

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/namespace"
)

var _ = Describe("Test MutatingWebhookConfiguration patch", func() {
	Context("find and patches webhook", func() {
		//cert := tresor.Certificate{}
		cert := mockCertificate{}
		meshName := "--meshName--"
		osmNamespace := "--namespace--"
		webhookName := "--webhookName--"
		//TODO:seed a test webhook
		testWebhookServiceNamespace := "test-namespace"
		testWebhookServiceName := "test-service-name"
		testWebhookServicePath := "/path"
		kubeClient := fake.NewSimpleClientset(&admissionv1beta1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: webhookName,
			},
			Webhooks: []admissionv1beta1.MutatingWebhook{
				{
					Name: osmWebhookName,
					ClientConfig: admissionv1beta1.WebhookClientConfig{
						Service: &admissionv1beta1.ServiceReference{
							Namespace: testWebhookServiceNamespace,
							Name:      testWebhookServiceName,
							Path:      &testWebhookServicePath,
						},
					},
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"some-key": "some-value",
						},
					},
				},
			},
		})

		It("checks if the hook exists", func() {
			err := hookExists(kubeClient, webhookName)
			Expect(err).ToNot(HaveOccurred())
		})

		It("checks if a non existent hook exists", func() {
			err := hookExists(kubeClient, webhookName+"blah")
			Expect(err).To(HaveOccurred())
		})

		It("patches a webhook", func() {
			err := patchMutatingWebhookConfiguration(cert, meshName, osmNamespace, webhookName, kubeClient)
			Expect(err).ToNot(HaveOccurred())

		})

		It("ensures webhook is configured correctly", func() {
			webhooks, err := kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(webhooks.Items)).To(Equal(1))

			webhook := webhooks.Items[0]
			Expect(len(webhook.Webhooks)).To(Equal(1))
			Expect(webhook.Webhooks[0].NamespaceSelector.MatchLabels["some-key"]).To(Equal("some-value"))
			Expect(webhook.Webhooks[0].ClientConfig.Service.Namespace).To(Equal(testWebhookServiceNamespace))
			Expect(webhook.Webhooks[0].ClientConfig.Service.Name).To(Equal(testWebhookServiceName))
			Expect(webhook.Webhooks[0].ClientConfig.Service.Path).To(Equal(&testWebhookServicePath))
			Expect(webhook.Webhooks[0].ClientConfig.CABundle).To(Equal([]byte("chain")))
			Expect(len(webhook.Webhooks[0].Rules)).To(Equal(1))
			rule := webhook.Webhooks[0].Rules[0]
			Expect(len(rule.Operations)).To(Equal(1))
			Expect(rule.Operations[0]).To(Equal(admissionv1beta1.Create))
			Expect(rule.Rule.APIGroups).To(Equal([]string{""}))
			Expect(rule.Rule.APIVersions).To(Equal([]string{"v1"}))
			Expect(rule.Rule.Resources).To(Equal([]string{"pods"}))
		})
	})
})

type mockCertificate struct{}

func (mc mockCertificate) GetCommonName() certificate.CommonName { return "" }
func (mc mockCertificate) GetCertificateChain() []byte           { return []byte("chain") }
func (mc mockCertificate) GetPrivateKey() []byte                 { return []byte("key") }
func (mc mockCertificate) GetIssuingCA() []byte                  { return []byte("ca") }
func (mc mockCertificate) GetExpiration() time.Time              { return time.Now() }

var _ = Describe("Testing isAnnotatedForInjection", func() {
	Context("when the inject annotation is one of enabled/yes/true", func() {
		It("should return true to enable sidecar injection", func() {
			annotation := map[string]string{constants.SidecarInjectionAnnotation: "enabled"}
			exists, enabled, err := isAnnotatedForInjection(annotation)
			Expect(exists).To(BeTrue())
			Expect(enabled).To(BeTrue())
			Expect(err).To(BeNil())
		})

		It("should return true to enable sidecar injection", func() {
			annotation := map[string]string{constants.SidecarInjectionAnnotation: "yes"}
			exists, enabled, err := isAnnotatedForInjection(annotation)
			Expect(exists).To(BeTrue())
			Expect(enabled).To(BeTrue())
			Expect(err).To(BeNil())
		})

		It("should return true to enable sidecar injection", func() {
			annotation := map[string]string{constants.SidecarInjectionAnnotation: "true"}
			exists, enabled, err := isAnnotatedForInjection(annotation)
			Expect(exists).To(BeTrue())
			Expect(enabled).To(BeTrue())
			Expect(err).To(BeNil())
		})
	})

	Context("when the inject annotation is one of disabled/no/false", func() {
		It("should return false to disable sidecar injection", func() {
			annotation := map[string]string{constants.SidecarInjectionAnnotation: "disabled"}
			exists, enabled, err := isAnnotatedForInjection(annotation)
			Expect(exists).To(BeTrue())
			Expect(enabled).To(BeFalse())
			Expect(err).To(BeNil())
		})

		It("should return false to disable sidecar injection", func() {
			annotation := map[string]string{constants.SidecarInjectionAnnotation: "no"}
			exists, enabled, err := isAnnotatedForInjection(annotation)
			Expect(exists).To(BeTrue())
			Expect(enabled).To(BeFalse())
			Expect(err).To(BeNil())
		})

		It("should return false to disable sidecar injection", func() {
			annotation := map[string]string{constants.SidecarInjectionAnnotation: "false"}
			exists, enabled, err := isAnnotatedForInjection(annotation)
			Expect(exists).To(BeTrue())
			Expect(enabled).To(BeFalse())
			Expect(err).To(BeNil())
		})
	})

	Context("when the inject annotation does not exist", func() {
		It("should return false to indicate the annotation does not exist", func() {
			annotation := map[string]string{}
			exists, _, _ := isAnnotatedForInjection(annotation)
			Expect(exists).To(BeFalse())
		})
	})

	Context("when an invalid inject annotation is specified", func() {
		It("should return an error", func() {
			annotation := map[string]string{constants.SidecarInjectionAnnotation: "invalid-value"}
			_, _, err := isAnnotatedForInjection(annotation)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Testing mustInject", func() {
	var (
		mockCtrl         *gomock.Controller
		mockNsController *namespace.MockController
		fakeClientSet    *fake.Clientset
		wh               *webhook
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockNsController = namespace.NewMockController(mockCtrl)
	fakeClientSet = fake.NewSimpleClientset()
	namespace := "test"

	BeforeEach(func() {
		fakeClientSet = fake.NewSimpleClientset()
		wh = &webhook{
			kubeClient:          fakeClientSet,
			namespaceController: mockNsController,
		}
	})
	AfterEach(func() {
		err := fakeClientSet.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return true when the pod is enabled for sidecar injection", func() {
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		podWithInjectAnnotationEnabled := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-with-injection-enabled",
				Annotations: map[string]string{
					constants.SidecarInjectionAnnotation: "enabled",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-SA",
			},
		}
		_, err = fakeClientSet.CoreV1().Pods(namespace).Create(context.TODO(), podWithInjectAnnotationEnabled, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		mockNsController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)

		inject, err := wh.mustInject(podWithInjectAnnotationEnabled, namespace)

		Expect(err).ToNot(HaveOccurred())
		Expect(inject).To(BeTrue())
	})

	It("should return false when the pod is disabled for sidecar injection", func() {
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		podWithInjectAnnotationEnabled := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-with-injection-disabled",
				Annotations: map[string]string{
					constants.SidecarInjectionAnnotation: "disabled",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-SA",
			},
		}
		_, err = fakeClientSet.CoreV1().Pods(namespace).Create(context.TODO(), podWithInjectAnnotationEnabled, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		mockNsController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)

		inject, err := wh.mustInject(podWithInjectAnnotationEnabled, namespace)

		Expect(err).ToNot(HaveOccurred())
		Expect(inject).To(BeFalse())
	})

	It("should return true when the namespace is enabled for injection and the pod is not explicitly disabled for injection", func() {
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Annotations: map[string]string{
					constants.SidecarInjectionAnnotation: "enabled",
				},
			},
		}
		_, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		podWithInjectAnnotationEnabled := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-with-no-injection-annotation",
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-SA",
			},
		}
		_, err = fakeClientSet.CoreV1().Pods(namespace).Create(context.TODO(), podWithInjectAnnotationEnabled, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		mockNsController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)

		inject, err := wh.mustInject(podWithInjectAnnotationEnabled, namespace)

		Expect(err).ToNot(HaveOccurred())
		Expect(inject).To(BeTrue())
	})

	It("should return false when the namespace is enabled for injection and the pod is explicitly disabled for injection", func() {
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Annotations: map[string]string{
					constants.SidecarInjectionAnnotation: "enabled",
				},
			},
		}
		_, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		podWithInjectAnnotationEnabled := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-with-injection-disabled",
				Annotations: map[string]string{
					constants.SidecarInjectionAnnotation: "disabled",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-SA",
			},
		}
		_, err = fakeClientSet.CoreV1().Pods(namespace).Create(context.TODO(), podWithInjectAnnotationEnabled, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		mockNsController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)

		inject, err := wh.mustInject(podWithInjectAnnotationEnabled, namespace)

		Expect(err).ToNot(HaveOccurred())
		Expect(inject).To(BeFalse())
	})

	It("should return false when the pod's namespace is not being monitored", func() {
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		podWithInjectAnnotationEnabled := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-with-injection-enabled",
				Annotations: map[string]string{
					constants.SidecarInjectionAnnotation: "enabled",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-SA",
			},
		}
		_, err = fakeClientSet.CoreV1().Pods(namespace).Create(context.TODO(), podWithInjectAnnotationEnabled, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		mockNsController.EXPECT().IsMonitoredNamespace(namespace).Return(false).Times(1)

		inject, err := wh.mustInject(podWithInjectAnnotationEnabled, namespace)

		Expect(err).ToNot(HaveOccurred())
		Expect(inject).To(BeFalse())
	})

	It("should return an error when an invalid annotation is specified", func() {
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		podWithInjectAnnotationEnabled := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-with-injection-enabled",
				Annotations: map[string]string{
					constants.SidecarInjectionAnnotation: "invalid-value",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-SA",
			},
		}
		_, err = fakeClientSet.CoreV1().Pods(namespace).Create(context.TODO(), podWithInjectAnnotationEnabled, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		mockNsController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)

		inject, err := wh.mustInject(podWithInjectAnnotationEnabled, namespace)

		Expect(err).To(HaveOccurred())
		Expect(inject).To(BeFalse())
	})
})
