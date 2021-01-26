package catalog

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

const (
	service1Name = "svc-name-1"
)

var _ = Describe("Test XDS certificate tooling", func() {
	mockCtrl := gomock.NewController(ginkgo.GinkgoT())
	kubeClient := testclient.NewSimpleClientset()

	mc := NewFakeMeshCatalog(kubeClient)
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", tests.ProxyUUID, tests.BookstoreServiceAccountName, tests.Namespace))

	Context("Test makeSyntheticServiceForPod()", func() {
		It("creates a MeshService struct with properly formatted Name and Namespace of the synthetic service", func() {
			namespace := uuid.New().String()
			serviceAccountName := uuid.New().String()
			cn := certificate.CommonName(uuid.New().String())
			pod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
				Spec: v1.PodSpec{
					ServiceAccountName: serviceAccountName,
				},
			}

			actual := makeSyntheticServiceForPod(pod, cn)

			expected := service.MeshService{
				Name:      fmt.Sprintf("%s.%s.osm.synthetic-%s", serviceAccountName, namespace, service.SyntheticServiceSuffix),
				Namespace: namespace,
			}
			Expect(len(actual)).To(Equal(1))
			Expect(actual[0]).To(Equal(expected))
		})
	})

	Context("Test GetServicesFromEnvoyCertificate()", func() {
		It("works as expected", func() {
			pod := tests.NewPodFixture(tests.Namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
			_, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.ServiceAccountName).To(Equal(tests.BookstoreServiceAccountName))

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, tests.Namespace, selector)
			_, err = kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			svcName2 := uuid.New().String()
			svc2 := tests.NewServiceFixture(svcName2, tests.Namespace, selector)
			_, err = kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), svc2, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			meshServices, err := mc.GetServicesFromEnvoyCertificate(cn)
			Expect(err).ToNot(HaveOccurred())
			expectedSvc := service.MeshService{
				Namespace: tests.Namespace,
				Name:      svcName,
			}

			expectedSvc2 := service.MeshService{
				Namespace: tests.Namespace,
				Name:      svcName2,
			}
			expectedList := []service.MeshService{expectedSvc, expectedSvc2}

			Expect(meshServices).To(Equal(expectedList))
		})

		It("returns an error with an invalid CN", func() {
			service, err := mc.GetServicesFromEnvoyCertificate("getAllowedDirectionalServices")
			Expect(err).To(HaveOccurred())
			Expect(service).To(BeNil())
		})
	})

	Context("Test getServiceFromCertificate()", func() {
		It("works as expected", func() {

			// Create the POD
			proxyUUID := uuid.New()
			namespace := uuid.New().String()
			podName := uuid.New().String()
			newPod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
			newPod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()

			_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, namespace, selector)
			_, err = kubeClient.CoreV1().Services(namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			podCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyUUID, tests.BookstoreServiceAccountName, namespace))
			meshServices, err := mc.GetServicesFromEnvoyCertificate(podCN)
			Expect(err).ToNot(HaveOccurred())

			expected := service.MeshService{
				Namespace: namespace,
				Name:      svcName,
			}
			expectedList := []service.MeshService{expected}

			Expect(meshServices).To(Equal(expectedList))
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("works as expected", func() {
			proxyUUID := uuid.New()
			someOtherEnvoyUID := uuid.New().String()
			namespace := uuid.New().String()
			mockKubeController := k8s.NewMockController(mockCtrl)
			podlabels := map[string]string{
				tests.SelectorKey:                tests.SelectorValue,
				constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
			}
			someOthePodLabels := map[string]string{
				tests.SelectorKey:                tests.SelectorValue,
				constants.EnvoyUniqueIDLabelName: someOtherEnvoyUID,
			}

			// Ensure correct presetup
			pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(0))

			newPod0 := tests.NewPodFixture(namespace, fmt.Sprintf("pod-0-%s", uuid.New()), tests.BookstoreServiceAccountName, someOthePodLabels)
			_, err = kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod0, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newPod1 := tests.NewPodFixture(namespace, fmt.Sprintf("pod-1-%s", uuid.New()), tests.BookstoreServiceAccountName, podlabels)
			_, err = kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod1, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newPod2 := tests.NewPodFixture(namespace, fmt.Sprintf("pod-2-%s", uuid.New()), tests.BookstoreServiceAccountName, someOthePodLabels)
			_, err = kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod2, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Ensure correct setup
			pods, err = kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(3))

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyUUID, tests.BookstoreServiceAccountName, namespace))

			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&newPod0, &newPod1, &newPod2})
			actualPod, err := GetPodFromCertificate(newCN, mockKubeController)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualPod.Name).To(Equal(newPod1.Name))
			Expect(actualPod).To(Equal(&newPod1))
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails with invalid certificate", func() {
			namespace := uuid.New().String()
			proxyUUID := uuid.New()
			mockKubeController := k8s.NewMockController(mockCtrl)

			// Create a pod with the same certificateCN twice
			for range []int{0, 1} {
				podName := uuid.New().String()
				newPod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
				newPod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()

				_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			// No service account in this CN
			newCN := certificate.CommonName(fmt.Sprintf("%s.%s", proxyUUID, namespace))
			actualPod, err := GetPodFromCertificate(newCN, mockKubeController)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errInvalidCertificateCN))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails with two pods with same cert", func() {
			namespace := uuid.New().String()
			proxyUUID := uuid.New()
			mockKubeController := k8s.NewMockController(mockCtrl)

			// Create a pod with the same certificateCN twice
			var pods []*v1.Pod
			for range []int{0, 1} {
				podName := uuid.New().String()
				tests.PodLabels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()
				newPod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
				pods = append(pods, &newPod)

				_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			mockKubeController.EXPECT().ListPods().Return(pods)
			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyUUID, tests.BookstoreServiceAccountName, namespace))
			actualPod, err := GetPodFromCertificate(newCN, mockKubeController)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errMoreThanOnePodForCertificate))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails when service account does not match certificate", func() {
			namespace := uuid.New().String()
			proxyUUID := uuid.New()
			randomServiceAccount := uuid.New().String()
			mockKubeController := k8s.NewMockController(mockCtrl)

			podName := uuid.New().String()
			newPod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
			newPod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()

			_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(newPod.Spec.ServiceAccountName).ToNot(Equal(randomServiceAccount))
			Expect(newPod.Spec.ServiceAccountName).To(Equal(tests.BookstoreServiceAccountName))

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyUUID, randomServiceAccount, namespace))
			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&newPod})
			actualPod, err := GetPodFromCertificate(newCN, mc.kubeController)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errServiceAccountDoesNotMatchCertificate))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails when namespace does not match certificate", func() {
			namespace := uuid.New().String()
			proxyUUID := uuid.New()
			someOtherRandomNamespace := uuid.New().String()
			mockKubeController := k8s.NewMockController(mockCtrl)

			podName := uuid.New().String()
			newPod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
			newPod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()

			_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyUUID, tests.BookstoreServiceAccountName, someOtherRandomNamespace))
			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&newPod})
			actualPod, err := GetPodFromCertificate(newCN, mockKubeController)
			Expect(err).To(HaveOccurred())
			// Since the namespace on the certificate is different than where the pod is...
			Expect(err).To(Equal(errDidNotFindPodForCertificate))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test listServicesForPod()", func() {
		It("lists services for pod", func() {
			namespace := uuid.New().String()
			selectors := map[string]string{tests.SelectorKey: tests.SelectorValue}
			mockKubeController := k8s.NewMockController(mockCtrl)
			var serviceNames []string
			var services []*v1.Service = []*v1.Service{}

			{
				// Create a service
				service := tests.NewServiceFixture(service1Name, namespace, selectors)
				services = append(services, service)
				_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceNames = append(serviceNames, service.Name)
			}

			{
				// Create a second service
				svc2Name := "svc-name-2"
				service := tests.NewServiceFixture(svc2Name, namespace, selectors)
				services = append(services, service)
				_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceNames = append(serviceNames, service.Name)
			}

			pod := tests.NewPodFixture(namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
			mockKubeController.EXPECT().ListServices().Return(services)
			actualSvcs, err := listServicesForPod(&pod, mockKubeController)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(actualSvcs)).To(Equal(2))

			actualNames := []string{actualSvcs[0].Name, actualSvcs[1].Name}
			Expect(actualNames).To(Equal(serviceNames))
		})

		It("should correctly not list services for pod that don't match the service's selectors", func() {
			namespace := uuid.New().String()
			selectors := map[string]string{"some-key": "some-value"}
			mockKubeController := k8s.NewMockController(mockCtrl)

			// Create a service
			service := tests.NewServiceFixture(service1Name, namespace, selectors)
			_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{service})
			pod := tests.NewPodFixture(namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
			actualSvcs, err := listServicesForPod(&pod, mockKubeController)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(actualSvcs)).To(Equal(0))
		})

		It("should correctly not list services for pod that don't match the service's selectors", func() {
			namespace := uuid.New().String()
			mockKubeController := k8s.NewMockController(mockCtrl)

			// The selector below has an additional label which the pod does not have.
			// Even though the first selector label matches the label on the pod, the
			// second selector label invalidates k8s selector matching criteria.
			selectors := map[string]string{
				tests.SelectorKey: tests.SelectorValue,
				"some-key":        "some-value",
			}

			// Create a service
			service := tests.NewServiceFixture(service1Name, namespace, selectors)
			_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{service})
			pod := tests.NewPodFixture(namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
			actualSvcs, err := listServicesForPod(&pod, mockKubeController)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(actualSvcs)).To(Equal(0))
		})
	})

	Context("Test getCertificateCommonNameMeta()", func() {
		It("parses CN into certificateCommonNameMeta", func() {
			proxyUUID := uuid.New()
			testNamespace := uuid.New().String()
			serviceAccount := uuid.New().String()

			cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyUUID, serviceAccount, testNamespace))

			cnMeta, err := getCertificateCommonNameMeta(cn)
			Expect(err).ToNot(HaveOccurred())

			expected := &certificateCommonNameMeta{
				ProxyUUID:      proxyUUID,
				ServiceAccount: serviceAccount,
				Namespace:      testNamespace,
			}
			Expect(cnMeta).To(Equal(expected))
		})

		It("parses CN into certificateCommonNameMeta", func() {
			_, err := getCertificateCommonNameMeta("a")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test NewCertCommonNameWithProxyID() and getCertificateCommonNameMeta() together", func() {
		It("returns the the CommonName of the form <proxyID>.<namespace>", func() {

			proxyUUID := uuid.New()
			serviceAccount := uuid.New().String()
			namespace := uuid.New().String()

			cn := NewCertCommonNameWithProxyID(proxyUUID, serviceAccount, namespace)
			Expect(cn).To(Equal(certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyUUID, serviceAccount, namespace))))

			actualMeta, err := getCertificateCommonNameMeta(cn)
			expectedMeta := certificateCommonNameMeta{
				ProxyUUID:      proxyUUID,
				ServiceAccount: serviceAccount,
				Namespace:      namespace,
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(actualMeta).To(Equal(&expectedMeta))
		})
	})

	Context("Test filterTrafficSplitServices()", func() {
		It("returns services except these to be traffic split", func() {

			services := []v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo",
						Name:      "A",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: tests.TrafficSplit.Namespace,
						Name:      tests.TrafficSplit.Spec.Service,
					},
				},
			}

			expected := []v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo",
						Name:      "A",
					},
				},
			}

			actual := mc.filterTrafficSplitServices(services)

			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test kubernetesServicesToMeshServices()", func() {
		It("converts a list of Kubernetes Services to a list of OSM Mesh Services", func() {

			services := []v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo",
						Name:      "A",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: tests.TrafficSplit.Namespace,
						Name:      tests.TrafficSplit.Spec.Service,
					},
				},
			}

			expected := []service.MeshService{
				{
					Namespace: "foo",
					Name:      "A",
				},
				{
					Namespace: "default",
					Name:      "bookstore-apex",
				},
			}

			actual := kubernetesServicesToMeshServices(services)

			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test listServiceNames()", func() {
		It("converts a list of OSM Mesh Services to a list of strings (the Mesh Service names)", func() {

			services := []service.MeshService{
				{
					Namespace: "foo",
					Name:      "A",
				},
				{
					Namespace: "default",
					Name:      "bookstore-apex",
				},
			}

			expected := []string{
				"foo/A",
				"default/bookstore-apex",
			}

			actual := listServiceNames(services)

			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test GetServiceAccountFromProxyCertificate", func() {
		It("should correctly return the ServiceAccount encoded in the XDS certificate CN", func() {
			cn := certificate.CommonName(fmt.Sprintf("%s.sa-name.sa-namespace", uuid.New().String()))
			svcAccount, err := GetServiceAccountFromProxyCertificate(cn)
			Expect(err).ToNot(HaveOccurred())
			Expect(svcAccount).To(Equal(service.K8sServiceAccount{Name: "sa-name", Namespace: "sa-namespace"}))
		})

		It("should correctly error when the XDS certificate CN is invalid", func() {
			svcAccount, err := GetServiceAccountFromProxyCertificate(certificate.CommonName("invalid"))
			Expect(err).To(HaveOccurred())
			Expect(svcAccount).To(Equal(service.K8sServiceAccount{}))
		})
	})
})
