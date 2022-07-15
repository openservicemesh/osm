package injector

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/webhook"
)

func TestCreateMutatingWebhook(t *testing.T) {
	assert := tassert.New(t)
	cert := &certificate.Certificate{
		CommonName:   "",
		CertChain:    pem.Certificate("chain"),
		PrivateKey:   pem.PrivateKey("key"),
		IssuingCA:    pem.RootCertificate("ca"),
		TrustedCAs:   pem.RootCertificate("ca"),
		Expiration:   time.Now(),
		SerialNumber: "serial_number",
	}
	webhookName := "--webhookName--"
	webhookTimeout := int32(20)
	meshName := "test-mesh"
	osmNamespace := "test-namespace"
	osmVersion := "test-version"
	webhookPath := webhookCreatePod
	webhookPort := int32(constants.InjectorWebhookPort)
	enableReconciler := true

	kubeClient := fake.NewSimpleClientset()
	err := createOrUpdateMutatingWebhook(kubeClient, cert, webhookTimeout, webhookName, meshName, osmNamespace, osmVersion, enableReconciler)
	assert.Nil(err)

	webhooks, err := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.TODO(), metav1.ListOptions{})
	assert.Nil(err)
	assert.Len(webhooks.Items, 1)

	wh := webhooks.Items[0]
	assert.Len(wh.Webhooks, 1)
	assert.Equal(wh.ObjectMeta.Name, webhookName)
	assert.EqualValues(wh.ObjectMeta.Labels, map[string]string{
		constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
		constants.OSMAppInstanceLabelKey: meshName,
		constants.OSMAppVersionLabelKey:  osmVersion,
		constants.AppLabel:               constants.OSMInjectorName,
		constants.ReconcileLabel:         strconv.FormatBool(true),
	})

	assert.Equal(wh.Webhooks[0].ClientConfig.Service.Namespace, osmNamespace)
	assert.Equal(wh.Webhooks[0].ClientConfig.Service.Name, constants.OSMInjectorName)
	assert.Equal(wh.Webhooks[0].ClientConfig.Service.Path, &webhookPath)
	assert.Equal(wh.Webhooks[0].ClientConfig.Service.Port, &webhookPort)
	assert.Equal(wh.Webhooks[0].ClientConfig.CABundle, []byte("ca"))

	assert.Equal(wh.Webhooks[0].NamespaceSelector.MatchLabels[constants.OSMKubeResourceMonitorAnnotation], meshName)
	assert.EqualValues(wh.Webhooks[0].NamespaceSelector.MatchExpressions, []metav1.LabelSelectorRequirement{
		{
			Key:      constants.IgnoreLabel,
			Operator: metav1.LabelSelectorOpDoesNotExist,
		},
		{
			Key:      "name",
			Operator: metav1.LabelSelectorOpNotIn,
			Values:   []string{osmNamespace},
		},
		{
			Key:      "control-plane",
			Operator: metav1.LabelSelectorOpDoesNotExist,
		},
	})
	assert.ElementsMatch(wh.Webhooks[0].Rules, []admissionregv1.RuleWithOperations{
		{
			Operations: []admissionregv1.OperationType{admissionregv1.Create},
			Rule: admissionregv1.Rule{
				APIGroups:   []string{"*"},
				APIVersions: []string{"v1"},
				Resources:   []string{"pods"},
			},
		},
	})
	assert.Equal(wh.Webhooks[0].TimeoutSeconds, &webhookTimeout)
	assert.Equal(wh.Webhooks[0].AdmissionReviewVersions, []string{"v1"})
}

func TestIsAnnotatedForInjection(t *testing.T) {
	testCases := []struct {
		name        string
		annotations map[string]string
		exists      bool
		enabled     bool
		expectError bool
	}{
		{
			name:        "annotation is set to enabled",
			annotations: map[string]string{constants.SidecarInjectionAnnotation: "enabled"},
			exists:      true,
			enabled:     true,
			expectError: false,
		},
		{
			name:        "annotation is set to yes",
			annotations: map[string]string{constants.SidecarInjectionAnnotation: "yes"},
			exists:      true,
			enabled:     true,
			expectError: false,
		},
		{
			name:        "annotation is set to true",
			annotations: map[string]string{constants.SidecarInjectionAnnotation: "true"},
			exists:      true,
			enabled:     true,
			expectError: false,
		},
		{
			name:        "annotation is set to disabled",
			annotations: map[string]string{constants.SidecarInjectionAnnotation: "disabled"},
			exists:      true,
			enabled:     false,
			expectError: false,
		},
		{
			name:        "annotation is set to no",
			annotations: map[string]string{constants.SidecarInjectionAnnotation: "no"},
			exists:      true,
			enabled:     false,
			expectError: false,
		},
		{
			name:        "annotation is set to false",
			annotations: map[string]string{constants.SidecarInjectionAnnotation: "false"},
			exists:      true,
			enabled:     false,
			expectError: false,
		},
		{
			name:        "annotation does not exist",
			annotations: map[string]string{},
			exists:      false,
			enabled:     false,
			expectError: false,
		},
		{
			name:        "annotation exists with an invalid value",
			annotations: map[string]string{constants.SidecarInjectionAnnotation: "invalid"},
			exists:      true,
			enabled:     false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actualExists, actualEnabled, actualErr := isAnnotatedForInjection(tc.annotations, "-kind-", "-name-")
			assert.Equal(tc.exists, actualExists)
			assert.Equal(tc.enabled, actualEnabled)
			assert.Equal(tc.expectError, actualErr != nil)
		})
	}
}

var _ = Describe("Testing mustInject, isNamespaceInjectable", func() {
	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		fakeClientSet      *fake.Clientset
		wh                 *mutatingWebhook
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	fakeClientSet = fake.NewSimpleClientset()
	namespace := "test"
	osmNamespace := "osm-namespace"

	BeforeEach(func() {
		fakeClientSet = fake.NewSimpleClientset()
		wh = &mutatingWebhook{
			kubeClient:     fakeClientSet,
			kubeController: mockKubeController,
			osmNamespace:   osmNamespace,
			nonInjectNamespaces: mapset.NewSetFromSlice([]interface{}{
				metav1.NamespaceSystem,
				metav1.NamespacePublic,
				osmNamespace,
			}),
		}
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

	It("should return false when the pod belongs to the host network", func() {
		p := &corev1.Pod{
			Spec: corev1.PodSpec{
				HostNetwork: true,
			},
		}

		inject, err := wh.mustInject(p, "")

		Expect(err).ToNot(HaveOccurred())
		Expect(inject).To(BeFalse())
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

	It("should return an error when isAnnotatedForInjection on namespace fails", func() {
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Annotations: map[string]string{
					constants.SidecarInjectionAnnotation: "invalid",
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

		Expect(err).To(HaveOccurred())
		Expect(inject).To(BeFalse())
	})

	It("Should allow a monitored app namespace", func() {
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		mockKubeController.EXPECT().IsMonitoredNamespace(testNamespace.Name).Return(true).Times(1)
		allowed := wh.isNamespaceInjectable(testNamespace.Name)
		Expect(allowed).To(BeTrue())
	})

	It("Should not allow an osm-controller's namespace", func() {
		allowed := wh.isNamespaceInjectable(osmNamespace)
		Expect(allowed).To(BeFalse())
	})

	It("Should not allow an kubernetes system namespace", func() {
		allowed := wh.isNamespaceInjectable(metav1.NamespaceSystem)
		Expect(allowed).To(BeFalse())
	})

	It("Should not allow an kubernetes public namespace", func() {
		allowed := wh.isNamespaceInjectable(metav1.NamespacePublic)
		Expect(allowed).To(BeFalse())
	})

	It("Should not allow an unmonitored app namespace", func() {
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err := fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		mockKubeController.EXPECT().IsMonitoredNamespace(testNamespace.Name).Return(false).Times(1)
		allowed := wh.isNamespaceInjectable(testNamespace.Name)
		Expect(allowed).To(BeFalse())
	})
})

var _ = Describe("Testing Injector Functions", func() {
	meshName := "-mesh-name-"
	osmNamespace := "-osm-namespace-"
	webhookName := "-webhook-name-"
	osmVersion := "-osm-version"
	enableReconciler := false
	webhookTimeout := int32(20)
	admissionRequestBody := `{
  "kind": "AdmissionReview",
  "apiVersion": "admission.k8s.io/v1",
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
}`
	It("creates new webhook", func() {
		kubeClient := fake.NewSimpleClientset(&admissionregv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: webhookName,
			},
		})
		var kubeController k8s.Controller
		stop := make(chan struct{})
		mockController := gomock.NewController(GinkgoT())
		cfg := configurator.NewMockConfigurator(mockController)
		certManager := tresorFake.NewFake(nil, 1*time.Hour)

		cfg.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()

		_, err := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), webhookName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		actualErr := NewMutatingWebhook(context.Background(), kubeClient, certManager, kubeController, meshName, osmNamespace, webhookName, osmVersion, webhookTimeout, enableReconciler, cfg, "")
		Expect(actualErr).NotTo(HaveOccurred())
		close(stop)
	})

	It("creates new webhook with reconciler enabled", func() {
		enableReconciler = true
		kubeClient := fake.NewSimpleClientset()
		var kubeController k8s.Controller
		stop := make(chan struct{})
		mockController := gomock.NewController(GinkgoT())
		cfg := configurator.NewMockConfigurator(mockController)
		certManager := tresorFake.NewFake(nil, 1*time.Hour)

		cfg.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()

		actualErr := NewMutatingWebhook(context.Background(), kubeClient, certManager, kubeController, meshName, osmNamespace, webhookName, osmVersion, webhookTimeout, enableReconciler, cfg, "")
		Expect(actualErr).NotTo(HaveOccurred())
		_, err := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), webhookName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		close(stop)
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
		wh := &mutatingWebhook{
			kubeClient:          client,
			kubeController:      mockNsController,
			nonInjectNamespaces: mapset.NewSet(),
		}

		req := httptest.NewRequest("GET", "/a/b/c", strings.NewReader(admissionRequestBody))
		req.Header = map[string][]string{
			"Content-Type": {"application/json"},
		}
		w := httptest.NewRecorder()
		mockNsController.EXPECT().IsMonitoredNamespace("default").Return(true).Times(1)
		wh.podCreationHandler(w, req)

		resp := w.Result()
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		expected := "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"11111111-2222-3333-4444-555555555555\",\"allowed\":true}}"
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		log.Debug().Msgf("Actual: %s", string(bodyBytes))
		Expect(string(bodyBytes)).To(Equal(expected))
	})

	It("getAdmissionReqResp creates admission ", func() {
		namespace := "default"
		client := fake.NewSimpleClientset()
		mockKubeController := k8s.NewMockController(gomock.NewController(GinkgoT()))
		mockKubeController.EXPECT().GetNamespace(namespace).Return(&corev1.Namespace{})
		mockKubeController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)

		wh := &mutatingWebhook{
			kubeClient:          client,
			kubeController:      mockKubeController,
			nonInjectNamespaces: mapset.NewSet(),
		}
		proxyUUID := uuid.New()

		// !! ACTION !!
		requestForNamespace, admissionResp := wh.getAdmissionReqResp(proxyUUID, []byte(admissionRequestBody))

		Expect(requestForNamespace).To(Equal("default"))

		expectedAdmissionResponse := admissionv1.AdmissionReview{
			TypeMeta: metav1.TypeMeta{Kind: "AdmissionReview", APIVersion: "admission.k8s.io/v1"},
			Request:  nil,
			Response: &admissionv1.AdmissionResponse{
				UID:              "11111111-2222-3333-4444-555555555555",
				Allowed:          true,
				Result:           nil,
				Patch:            nil,
				PatchType:        nil,
				AuditAnnotations: nil,
			},
		}
		Expect(admissionResp).To(Equal(expectedAdmissionResponse))
	})

	It("getAdmissionReqResp creates admission with error when decoding body fails", func() {
		namespace := "default"
		client := fake.NewSimpleClientset()
		mockKubeController := k8s.NewMockController(gomock.NewController(GinkgoT()))
		mockKubeController.EXPECT().GetNamespace(namespace).Return(&corev1.Namespace{})
		mockKubeController.EXPECT().IsMonitoredNamespace(namespace).Return(true).Times(1)

		wh := &mutatingWebhook{
			kubeClient:          client,
			kubeController:      mockKubeController,
			nonInjectNamespaces: mapset.NewSet(),
		}
		proxyUUID := uuid.New()

		// !! ACTION !!
		requestForNamespace, admissionResp := wh.getAdmissionReqResp(proxyUUID, []byte("}"))

		Expect(requestForNamespace).To(Equal(""))

		expectedAdmissionResponse := webhook.AdmissionError(fmt.Errorf("yaml: did not find expected node content"))
		Expect(admissionResp.Response).To(Equal(expectedAdmissionResponse))
	})

	It("mutate() handles nil admission request", func() {
		client := fake.NewSimpleClientset()
		mockNsController := k8s.NewMockController(gomock.NewController(GinkgoT()))
		mockNsController.EXPECT().GetNamespace("default").Return(&corev1.Namespace{})
		wh := &mutatingWebhook{
			kubeClient:          client,
			kubeController:      mockNsController,
			nonInjectNamespaces: mapset.NewSet(),
		}
		proxyUUID := uuid.New()

		// Action !!
		actual := wh.mutate(nil, proxyUUID)

		expected := admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: "nil admission request",
			},
		}
		Expect(actual).To(Equal(&expected))
	})

	It("patches admission response", func() {
		admRes := admissionv1.AdmissionResponse{
			Patch: []byte(""),
		}
		patchBytes := []byte("abc")
		patchAdmissionResponse(&admRes, patchBytes)

		expectedPatchType := admissionv1.PatchTypeJSONPatch
		expected := admissionv1.AdmissionResponse{
			Patch:     []byte("abc"),
			PatchType: &expectedPatchType,
		}
		Expect(admRes).To(Equal(expected))
	})
})

func TestPodCreationHandler(t *testing.T) {
	tests := []struct {
		name                 string
		req                  *http.Request
		expectedResponseCode int
	}{
		{
			name: "bad content-type",
			req: &http.Request{
				URL: &url.URL{
					RawQuery: "timeout=1s",
				},
				Header: http.Header{
					"Content-Type": []string{"bad"},
				},
			},
			expectedResponseCode: http.StatusUnsupportedMediaType,
		},
		{
			name: "error getting admission request body",
			req: &http.Request{
				URL: &url.URL{
					RawQuery: "timeout=1s",
				},
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			},
			expectedResponseCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)

			var wh *mutatingWebhook
			w := httptest.NewRecorder()

			wh.podCreationHandler(w, test.req)
			res := w.Result()
			if test.expectedResponseCode > 0 {
				assert.Equal(test.expectedResponseCode, res.StatusCode)
			}
		})
	}
}

func TestWebhookMutate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigurator.EXPECT().GetEnvoyImage().Return("envoy-linux-image").AnyTimes()
	mockConfigurator.EXPECT().GetEnvoyWindowsImage().Return("envoy-windows-image").AnyTimes()
	mockConfigurator.EXPECT().GetInitContainerImage().Return("init-container-image").AnyTimes()

	t.Run("invalid JSON", func(t *testing.T) {
		wh := &mutatingWebhook{
			configurator: mockConfigurator,
		}
		req := &admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{Raw: []byte("{")},
		}
		expected := "unexpected end of JSON input"
		res := wh.mutate(req, uuid.New())
		tassert.Contains(t, res.Result.Message, expected)
	})

	t.Run("mustInject error", func(t *testing.T) {
		assert := tassert.New(t)

		namespace := "ns"

		mockCtrl := gomock.NewController(t)
		kubeController := k8s.NewMockController(mockCtrl)
		kubeController.EXPECT().GetNamespace(namespace).Return(nil)
		kubeController.EXPECT().IsMonitoredNamespace(namespace).Return(true)
		wh := &mutatingWebhook{
			nonInjectNamespaces: mapset.NewSet(),
			kubeController:      kubeController,
			configurator:        mockConfigurator,
		}

		req := &admissionv1.AdmissionRequest{
			Namespace: namespace,
			Object: runtime.RawExtension{
				Raw: []byte(`{
					"apiVersion": "v1",
					"kind": "Pod"
				}`),
			},
		}

		res := wh.mutate(req, uuid.New())
		assert.Equal(res.Result.Message, errNamespaceNotFound.Error())
	})

	t.Run("createPatch error", func(t *testing.T) {
		assert := tassert.New(t)

		namespace := "ns"

		mockCtrl := gomock.NewController(t)
		kubeController := k8s.NewMockController(mockCtrl)
		kubeController.EXPECT().GetNamespace(namespace).Return(&corev1.Namespace{}).Times(1)
		kubeController.EXPECT().GetNamespace(namespace).Return(nil).Times(1)
		kubeController.EXPECT().IsMonitoredNamespace(namespace).Return(true)

		cfg := configurator.NewMockConfigurator(mockCtrl)
		cfg.EXPECT().GetMeshConfig().AnyTimes()
		cfg.EXPECT().GetInitContainerImage().Return("init-container-image").AnyTimes()
		cfg.EXPECT().GetEnvoyImage().Return("envoy-linux-image").AnyTimes()
		cfg.EXPECT().GetEnvoyWindowsImage().Return("envoy-windows-image").AnyTimes()

		wh := &mutatingWebhook{
			nonInjectNamespaces: mapset.NewSet(),
			kubeController:      kubeController,
			certManager:         tresorFake.NewFake(nil, 1*time.Hour),
			kubeClient:          fake.NewSimpleClientset(),
			configurator:        cfg,
		}

		req := &admissionv1.AdmissionRequest{
			Namespace: namespace,
			Object: runtime.RawExtension{
				Raw: []byte(`{
					"apiVersion": "v1",
					"kind": "Pod",
					"metadata": {
						"annotations": {
							"openservicemesh.io/sidecar-injection": "true"
						}
					}
				}`),
			},
		}

		res := wh.mutate(req, uuid.New())
		assert.Contains(res.Result.Message, errNamespaceNotFound.Error())
	})

	t.Run("will inject", func(t *testing.T) {
		assert := tassert.New(t)

		namespace := "ns"

		mockCtrl := gomock.NewController(t)
		kubeController := k8s.NewMockController(mockCtrl)
		kubeController.EXPECT().GetNamespace(namespace).Return(&corev1.Namespace{}).Times(2)
		kubeController.EXPECT().IsMonitoredNamespace(namespace).Return(true)

		cfg := configurator.NewMockConfigurator(mockCtrl)
		cfg.EXPECT().GetMeshConfig().AnyTimes()
		cfg.EXPECT().IsPrivilegedInitContainer()
		cfg.EXPECT().GetInitContainerImage().Return("init-container-image").AnyTimes()
		cfg.EXPECT().GetEnvoyImage().Return("envoy-linux-image").AnyTimes()
		cfg.EXPECT().GetEnvoyWindowsImage().Return("envoy-windows-image").AnyTimes()
		cfg.EXPECT().GetProxyResources()
		cfg.EXPECT().GetEnvoyLogLevel()

		wh := &mutatingWebhook{
			nonInjectNamespaces: mapset.NewSet(),
			kubeController:      kubeController,
			certManager:         tresorFake.NewFake(nil, 1*time.Hour),
			kubeClient:          fake.NewSimpleClientset(),
			configurator:        cfg,
		}

		req := &admissionv1.AdmissionRequest{
			Namespace: namespace,
			Object: runtime.RawExtension{
				Raw: []byte(`{
					"apiVersion": "v1",
					"kind": "Pod",
					"metadata": {
						"annotations": {
							"openservicemesh.io/sidecar-injection": "true"
						}
					}
				}`),
			},
		}

		res := wh.mutate(req, uuid.New())
		assert.NotNil(res.Patch)
	})
}
