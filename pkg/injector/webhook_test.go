package injector

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/google/uuid"
	"k8s.io/api/admission/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

var _ = Describe("Test MutatingWebhookConfiguration patch", func() {
	Context("find and patches webhook", func() {
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
					Name: mutatingWebhookName,
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
			exists, enabled, err := isAnnotatedForInjection(annotation)
			Expect(exists).To(BeFalse())
			Expect(enabled).To(BeFalse())
			Expect(err).To(BeNil())
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

var _ = Describe("Testing mustInject, isNamespaceAllowed", func() {
	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		fakeClientSet      *fake.Clientset
		wh                 *webhook
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	fakeClientSet = fake.NewSimpleClientset()
	namespace := "test"

	BeforeEach(func() {
		fakeClientSet = fake.NewSimpleClientset()
		wh = &webhook{
			kubeClient:     fakeClientSet,
			kubeController: mockKubeController,
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
		retNs, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
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

		mockKubeController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)
		mockKubeController.EXPECT().GetNamespace(namespace).Return(retNs)

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
		retNs, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
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

		mockKubeController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)
		mockKubeController.EXPECT().GetNamespace(namespace).Return(retNs)

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
		retNs, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
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

		mockKubeController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)
		mockKubeController.EXPECT().GetNamespace(namespace).Return(retNs)

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
		retNs, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
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

		mockKubeController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)
		mockKubeController.EXPECT().GetNamespace(namespace).Return(retNs)

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

		mockKubeController.EXPECT().IsMonitoredNamespace(namespace).Return(false).Times(1)

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

		mockKubeController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)

		inject, err := wh.mustInject(podWithInjectAnnotationEnabled, namespace)

		Expect(err).To(HaveOccurred())
		Expect(inject).To(BeFalse())
	})
})

var _ = Describe("Testing Injector Functions", func() {
	It("creates new webhook", func() {
		injectorConfig := Config{
			InitContainerImage: "-testInitContainerImage-",
			SidecarImage:       "-testSidecarImage-",
		}
		kubeClient := fake.NewSimpleClientset()
		var meshCatalog catalog.MeshCataloger
		var kubeController k8s.Controller
		meshName := "-mesh-name-"
		osmNamespace := "-osm-namespace-"
		webhookName := "-webhook-name-"
		stop := make(<-chan struct{})
		mockController := gomock.NewController(GinkgoT())
		cfg := configurator.NewMockConfigurator(mockController)
		cache := make(map[certificate.CommonName]certificate.Certificater)
		certManager := tresor.NewFakeCertManager(&cache, cfg)

		actualErr := NewWebhook(injectorConfig, kubeClient, certManager, meshCatalog, kubeController, meshName, osmNamespace, webhookName, stop, cfg)
		expectedErrorMessage := "Error configuring MutatingWebhookConfiguration: mutatingwebhookconfigurations.admissionregistration.k8s.io \"-webhook-name-\" not found"
		Expect(actualErr.Error()).To(Equal(expectedErrorMessage))
	})

	It("creates new webhook", func() {
		client := fake.NewSimpleClientset()
		mockNsController := k8s.NewMockController(gomock.NewController(GinkgoT()))
		mockNsController.EXPECT().GetNamespace("default").Return(&corev1.Namespace{})
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		}
		_, err := client.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		wh := &webhook{
			kubeClient:     client,
			kubeController: mockNsController,
		}
		body := strings.NewReader(`{
  "kind": "AdmissionReview",
  "apiVersion": "admission.k8s.io/v1beta1",
  "request": {
    "uid": "11111111-2222-3333-4444-555555555555",
    "kind": {
      "group": "",
      "version": "v1",
      "kind": "PodExecOptions"
    },
    "resource": {
      "group": "",
      "version": "v1",
      "resource": "pods"
    },
    "subResource": "exec",
    "requestKind": {
      "group": "",
      "version": "v1",
      "kind": "PodExecOptions"
    },
    "requestResource": {
      "group": "",
      "version": "v1",
      "resource": "pods"
    },
    "requestSubResource": "exec",
    "name": "some-pod-1111111111-22222",
    "namespace": "default",
    "operation": "CONNECT",
    "userInfo": {
      "username": "user",
      "groups": []
    },
    "object": {
      "kind": "PodExecOptions",
      "apiVersion": "v1",
      "stdin": true,
      "stdout": true,
      "tty": true,
      "container": "some-pod",
      "command": ["bin/bash"]
    },
    "oldObject": null,
    "dryRun": false,
    "options": null
  }
}`)
		req := httptest.NewRequest("GET", "/a/b/c", body)
		req.Header = map[string][]string{
			"Content-Type": {"application/json"},
		}
		w := httptest.NewRecorder()
		mockNsController.EXPECT().IsMonitoredNamespace("default").Return(true).Times(1)
		wh.mutateHandler(w, req)

		resp := w.Result()
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		expected := "{\"response\":{\"uid\":\"11111111-2222-3333-4444-555555555555\",\"allowed\":true}}"
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(string(bodyBytes)).To(Equal(expected))
	})

	It("handles health requests", func() {
		client := fake.NewSimpleClientset()
		mockNsController := k8s.NewMockController(gomock.NewController(GinkgoT()))
		mockNsController.EXPECT().GetNamespace("default").Return(&corev1.Namespace{})
		wh := &webhook{
			kubeClient:     client,
			kubeController: mockNsController,
		}
		w := httptest.NewRecorder()
		body := strings.NewReader(``)
		req := httptest.NewRequest("GET", "/a/b/c", body)

		// Action !!
		wh.healthHandler(w, req)

		resp := w.Result()
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		expected := "Health OK"
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(string(bodyBytes)).To(Equal(expected))
	})

	It("mutate() handles nil admission request", func() {
		client := fake.NewSimpleClientset()
		mockNsController := k8s.NewMockController(gomock.NewController(GinkgoT()))
		mockNsController.EXPECT().GetNamespace("default").Return(&corev1.Namespace{})
		wh := &webhook{
			kubeClient:     client,
			kubeController: mockNsController,
		}

		// Action !!
		actual := wh.mutate(nil)

		expected := v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: "nil admission request",
			},
		}
		Expect(actual).To(Equal(&expected))
	})

	It("patches admission response", func() {
		admRes := v1beta1.AdmissionResponse{
			Patch: []byte(""),
		}
		patchBytes := []byte("abc")
		patchAdmissionResponse(&admRes, patchBytes)

		expectedPatchType := v1beta1.PatchTypeJSONPatch
		expected := v1beta1.AdmissionResponse{
			Patch:     []byte("abc"),
			PatchType: &expectedPatchType,
		}
		Expect(admRes).To(Equal(expected))
	})

	It("creates admission error", func() {
		message := uuid.New().String()
		err := errors.New(message)
		actual := toAdmissionError(err)

		expected := v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: message,
			},
		}
		Expect(actual).To(Equal(&expected))
	})
})
