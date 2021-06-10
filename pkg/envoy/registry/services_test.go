package registry

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

const (
	service1Name = "svc-name-1"
)

var _ = Describe("Test Proxy-Service mapping", func() {
	mockCtrl := gomock.NewController(ginkgo.GinkgoT())
	kubeClient := testclient.NewSimpleClientset()
	mockKubeController := k8s.NewMockController(mockCtrl)
	proxyRegistry := NewProxyRegistry(&KubeProxyServiceMapper{mockKubeController})

	Context("Test ListProxyServices()", func() {
		It("works as expected", func() {
			pod := tests.NewPodFixture(tests.Namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
			Expect(pod.Spec.ServiceAccountName).To(Equal(tests.BookstoreServiceAccountName))
			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&pod}).Times(1)

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, tests.Namespace, selector)

			svcName2 := uuid.New().String()
			svc2 := tests.NewServiceFixture(svcName2, tests.Namespace, selector)
			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{svc, svc2}).Times(1)

			certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", tests.ProxyUUID, envoy.KindSidecar, tests.BookstoreServiceAccountName, tests.Namespace))
			certSerialNumber := certificate.SerialNumber("123456")
			proxy, err := envoy.NewProxy(certCommonName, certSerialNumber, nil)
			Expect(err).ToNot(HaveOccurred())
			meshServices, err := proxyRegistry.ListProxyServices(proxy)
			Expect(err).ToNot(HaveOccurred())
			expectedSvc := service.MeshService{
				Namespace:     tests.Namespace,
				Name:          svcName,
				ClusterDomain: constants.ClusterDomain,
			}

			expectedSvc2 := service.MeshService{
				Namespace:     tests.Namespace,
				Name:          svcName2,
				ClusterDomain: constants.ClusterDomain,
			}
			expectedList := []service.MeshService{expectedSvc, expectedSvc2}

			Expect(meshServices).Should(HaveLen(len(expectedList)))
			Expect(meshServices).Should(ConsistOf(expectedList))
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

			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&newPod}).Times(1)

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, namespace, selector)
			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{svc}).Times(1)

			podCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", proxyUUID, envoy.KindSidecar, tests.BookstoreServiceAccountName, namespace))
			certSerialNumber := certificate.SerialNumber("123456")
			newProxy, err := envoy.NewProxy(podCN, certSerialNumber, nil)
			Expect(err).ToNot(HaveOccurred())
			meshServices, err := proxyRegistry.ListProxyServices(newProxy)
			Expect(err).ToNot(HaveOccurred())

			expected := service.MeshService{
				Namespace:     namespace,
				Name:          svcName,
				ClusterDomain: constants.ClusterDomain,
			}
			expectedList := []service.MeshService{expected}

			Expect(meshServices).To(Equal(expectedList))
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
})
