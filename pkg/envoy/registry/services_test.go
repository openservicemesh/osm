package registry

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
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
	proxyRegistry := NewProxyRegistry(&KubeProxyServiceMapper{mockKubeController}, nil)

	Context("Test ListProxyServices()", func() {
		It("works as expected", func() {
			proxyUUID := uuid.New()

			podName := "pod-name"
			podName2 := "pod-name-2"
			pod := tests.NewPodFixture(tests.Namespace, podName, tests.BookstoreServiceAccountName,
				map[string]string{
					constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
					constants.AppLabel:               tests.SelectorValue})
			Expect(pod.Spec.ServiceAccountName).To(Equal(tests.BookstoreServiceAccountName))
			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{pod}).Times(1)

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{constants.AppLabel: tests.SelectorValue}
			svc1 := tests.NewServiceFixture(svcName, tests.Namespace, selector)

			svcName2 := uuid.New().String()
			svc2 := tests.HeadlessSvc(tests.NewServiceFixture(svcName2, tests.Namespace, selector))
			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{svc1, svc2}).Times(1)

			expectedSvc1 := service.MeshService{
				Namespace: tests.Namespace,
				Name:      svcName,
				Port:      tests.ServicePort,
				Protocol:  "http",
			}

			expectedSvc2 := service.MeshService{
				Namespace: tests.Namespace,
				Name:      fmt.Sprintf("%s.%s", podName, svcName2),
				Port:      tests.ServicePort,
				Protocol:  "http",
			}

			expectedList := []service.MeshService{expectedSvc1, expectedSvc2}

			mockKubeController.EXPECT().GetEndpoints(gomock.Any()).Return(&v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: tests.Namespace,
					Name:      expectedSvc1.Name,
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{
								IP: "8.8.8.8", // pod IP
							},
							{
								IP: "8.8.8.9", // pod2 IP
							},
						},
					},
				},
			}, nil).Times(1)

			mockKubeController.EXPECT().GetEndpoints(gomock.Any()).Return(&v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: tests.Namespace,
					Name:      expectedSvc2.Name,
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{
								IP:       "8.8.8.9", // pod IP
								Hostname: podName,
							},
							{
								IP:       "8.8.8.9", // pod2 IP
								Hostname: podName2,
							},
						},
					},
				},
			}, nil).Times(1)

			proxy := envoy.NewProxy(envoy.KindSidecar, proxyUUID, identity.New(tests.BookstoreServiceAccountName, tests.Namespace), nil)
			mockKubeController.EXPECT().GetPodForProxy(proxy).Return(pod, nil).Times(1)
			meshServices, err := proxyRegistry.ListProxyServices(proxy)
			Expect(err).ToNot(HaveOccurred())

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

			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{newPod}).Times(1)

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{constants.AppLabel: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, namespace, selector)
			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{svc}).Times(1)

			newProxy := envoy.NewProxy(envoy.KindSidecar, proxyUUID, identity.New(tests.BookstoreServiceAccountName, namespace), nil)

			expected := service.MeshService{
				Namespace: namespace,
				Name:      svcName,
				Port:      tests.ServicePort,
				Protocol:  "http",
			}
			expectedList := []service.MeshService{expected}
			mockKubeController.EXPECT().GetEndpoints(gomock.Any()).Return(nil, nil)

			mockKubeController.EXPECT().GetPodForProxy(newProxy).Return(newPod, nil).Times(1)
			// Subdomain gets called in the ListProxyServices
			meshServices, err := proxyRegistry.ListProxyServices(newProxy)
			Expect(err).ToNot(HaveOccurred())

			Expect(meshServices).To(Equal(expectedList))
		})
	})

	Context("Test listServicesForPod()", func() {
		It("lists services for pod", func() {
			namespace := uuid.New().String()
			selectors := map[string]string{constants.AppLabel: tests.SelectorValue}
			mockKubeController := k8s.NewMockController(mockCtrl)
			var serviceNames []string
			var services []*v1.Service = []*v1.Service{}
			svc2Name := "svc-name-2-headless"
			podName := "pod-name"

			{
				// Create a service
				service := tests.NewServiceFixture(service1Name, namespace, selectors)
				services = append(services, service)
				_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceNames = append(serviceNames, service.Name)
			}

			{
				// Create a second (headless) service
				service := tests.HeadlessSvc(tests.NewServiceFixture(svc2Name, namespace, selectors))
				services = append(services, service)
				_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceNames = append(serviceNames, fmt.Sprintf("%s.%s", podName, service.Name))
			}

			pod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
			mockKubeController.EXPECT().ListServices().Return(services)
			mockKubeController.EXPECT().GetEndpoints(gomock.Any()).Return(&v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      service1Name,
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{
								IP: "8.8.8.8",
							},
						},
					},
				},
			}, nil).Times(1)

			mockKubeController.EXPECT().GetEndpoints(gomock.Any()).Return(&v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: tests.Namespace,
					Name:      svc2Name,
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{
								IP:       "8.8.8.8",
								Hostname: pod.Name,
							},
						},
					},
				},
			}, nil).Times(1)
			actualSvcs := listServicesForPod(pod, mockKubeController)
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
			actualSvcs := listServicesForPod(pod, mockKubeController)
			Expect(len(actualSvcs)).To(Equal(0))
		})

		It("should correctly not list services for pod that don't match the service's selectors", func() {
			namespace := uuid.New().String()
			mockKubeController := k8s.NewMockController(mockCtrl)

			// The selector below has an additional label which the pod does not have.
			// Even though the first selector label matches the label on the pod, the
			// second selector label invalidates k8s selector matching criteria.
			selectors := map[string]string{
				constants.AppLabel: tests.SelectorValue,
				"some-key":         "some-value",
			}

			// Create a service
			service := tests.NewServiceFixture(service1Name, namespace, selectors)
			_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{service})
			pod := tests.NewPodFixture(namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
			actualSvcs := listServicesForPod(pod, mockKubeController)
			Expect(len(actualSvcs)).To(Equal(0))
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

func TestKubernetesServicesToMeshServices(t *testing.T) {
	testCases := []struct {
		name                 string
		k8sServices          []v1.Service
		k8sEndpoints         v1.Endpoints
		expectedMeshServices []service.MeshService
		subdomainFilter      string
	}{
		{
			name: "k8s services to mesh services",
			k8sServices: []v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "10.0.0.1",
						Ports: []v1.ServicePort{{
							Name: "p1",
							Port: 80,
						}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns2",
						Name:      "s2",
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "10.0.0.1",
						Ports: []v1.ServicePort{{
							Name: "p2",
							Port: 80,
						}},
					},
				},
			},
			expectedMeshServices: []service.MeshService{
				{
					Namespace: "ns1",
					Name:      "s1",
					Protocol:  "http",
					Port:      80,
				},
				{
					Namespace: "ns2",
					Name:      "s2",
					Protocol:  "http",
					Port:      80,
				},
			},
		},
		{
			name: "k8s services to mesh services (subdomain filter)",
			k8sServices: []v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1-headless",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name: "p1",
							Port: 80,
						}},
					},
				},
			},
			k8sEndpoints: v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1-headless",
					Namespace: "ns1",
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{
								IP:       "8.8.8.8",
								Hostname: "pod-0",
							},
							{
								IP:       "8.8.8.8",
								Hostname: "pod-1",
							},
						},
					},
				},
			},
			subdomainFilter: "pod-1",
			expectedMeshServices: []service.MeshService{
				{
					Namespace: "ns1",
					Name:      "pod-1.s1-headless",
					Protocol:  "http",
					Port:      80,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubeController := k8s.NewMockController(mockCtrl)

			mockKubeController.EXPECT().GetEndpoints(gomock.Any()).Return(&tc.k8sEndpoints, nil).Times(len(tc.k8sServices))

			actual := kubernetesServicesToMeshServices(mockKubeController, tc.k8sServices, tc.subdomainFilter)
			assert.ElementsMatch(tc.expectedMeshServices, actual)
		})
	}
}
