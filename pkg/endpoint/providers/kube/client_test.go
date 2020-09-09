package kube

import (
	"context"
	"net"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/utils"
)

var _ = Describe("Test Kube Client Provider", func() {
	var (
		mockCtrl         *gomock.Controller
		mockNsController *k8s.MockController
		mockConfigurator *configurator.MockConfigurator
	)
	mockCtrl = gomock.NewController(GinkgoT())
	mockNsController = k8s.NewMockController(mockCtrl)
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	fakeClientSet := fake.NewSimpleClientset()
	stopChann := make(chan struct{})
	providerID := "provider"

	cli, err := NewProvider(fakeClientSet, mockNsController, stopChann, providerID, mockConfigurator)

	mockNsController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()
	Context("when testing ListEndpointsForService", func() {
		It("verifies new Provider at context scope succeeded", func() {
			Expect(err).To(BeNil())
		})

		It("tests GetID", func() {
			Expect(cli.GetID()).To(Equal(providerID))
		})

		It("should correctly return a list of endpoints for a service", func() {
			// Should be empty for now
			Expect(cli.ListEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{}))

			// Create bookbuyer endpoint in Bookbuyer namespace
			endp := &corev1.Endpoints{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Endpoints",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: tests.BookbuyerService.Name,
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{
								IP: "8.8.8.8",
							},
						},
						Ports: []v1.EndpointPort{
							{
								Name:     "port",
								Port:     88,
								Protocol: v1.ProtocolTCP,
							},
						},
					},
				},
			}

			fakeClientSet.CoreV1().Endpoints(tests.BookbuyerService.Namespace).
				Create(context.TODO(), endp, metav1.CreateOptions{})

			<-cli.GetAnnouncementsChannel()
			Expect(cli.ListEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{
				{
					IP:   net.IPv4(8, 8, 8, 8),
					Port: 88,
				},
			}))
		})
	})

	It("tests GetAnnouncementChannel", func() {
		ch := cli.GetAnnouncementsChannel()

		// Can only expect to not be null
		Expect(ch).ToNot(BeNil())
	})

	Context("Testing FakeProvider", func() {
		It("returns empty list", func() {
			c := NewFakeProvider()
			svc := service.MeshService{
				Name: "blah",
			}
			Ω(func() { c.ListEndpointsForService(svc) }).Should(Panic())
		})

		It("returns the services for a given service account", func() {
			c := NewFakeProvider()
			actual := c.ListEndpointsForService(tests.BookstoreService)
			expected := []endpoint.Endpoint{{
				IP:   net.ParseIP("8.8.8.8"),
				Port: 8888,
			}}
			Expect(actual).To(Equal(expected))
		})
		It("Testing GetServicesForServiceAccount", func() {
			c := NewFakeProvider()

			sMesh, err := c.GetServicesForServiceAccount(tests.BookstoreServiceAccount)
			Expect(err).To(BeNil())
			expectedServices := []service.MeshService{tests.BookstoreService, tests.BookstoreApexService}
			Expect(sMesh).To(Equal(expectedServices))

			sMesh2, err := c.GetServicesForServiceAccount(tests.BookbuyerServiceAccount)
			Expect(err).To(BeNil())
			expectedServices2 := []service.MeshService{tests.BookbuyerService}
			Expect(sMesh2).To(Equal(expectedServices2))
		})
	})
})

var _ = Describe("When getting a Service associated with a ServiceAccount", func() {
	var (
		mockCtrl         *gomock.Controller
		mockNsController *k8s.MockController
		mockConfigurator *configurator.MockConfigurator
	)
	mockCtrl = gomock.NewController(GinkgoT())
	mockNsController = k8s.NewMockController(mockCtrl)
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	var (
		fakeClientSet *fake.Clientset
		provider      endpoint.Provider
		err           error
	)

	providerID := "test-provider"
	testNamespace := "test"
	stop := make(chan struct{})

	// Configure the controller
	listMonitoredNs := []string{
		testNamespace,
	}
	mockNsController.EXPECT().ListServices().DoAndReturn(func() []*corev1.Service {
		var services []*corev1.Service

		for _, ns := range listMonitoredNs {
			// simulate lookup on controller cache
			svcList, _ := fakeClientSet.CoreV1().Services(ns).List(context.TODO(), metav1.ListOptions{})
			for serviceIdx := range svcList.Items {
				services = append(services, &svcList.Items[serviceIdx])
			}
		}

		return services
	}).AnyTimes()
	mockNsController.EXPECT().GetService(gomock.Any()).DoAndReturn(func(msh service.MeshService) *v1.Service {
		// simulate lookup on controller cache
		vv, err := fakeClientSet.CoreV1().Services(msh.Namespace).Get(context.TODO(), msh.Name, metav1.GetOptions{})

		if err != nil {
			return nil
		}

		return vv
	}).AnyTimes()

	mockNsController.EXPECT().IsMonitoredNamespace(testNamespace).Return(true).AnyTimes()

	BeforeEach(func() {
		fakeClientSet = fake.NewSimpleClientset()
		provider, err = NewProvider(fakeClientSet, mockNsController, stop, providerID, mockConfigurator)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return not return a service when a Deployment matching the selector doesn't exist", func() {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: testNamespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Name:     "servicePort",
					Protocol: corev1.ProtocolTCP,
					Port:     tests.ServicePort,
				}},
				Selector: map[string]string{
					"app": "test",
				},
			},
		}

		_, err := fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			_, err := provider.GetServicesForServiceAccount(tests.BookbuyerServiceAccount)
			return err
		}, 2*time.Second).Should(HaveOccurred())

		err = fakeClientSet.CoreV1().Services(testNamespace).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return a service that matches the ServiceAccount associated with the Deployment", func() {
		// Create a Service
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: testNamespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Name:     "servicePort",
					Protocol: corev1.ProtocolTCP,
					Port:     tests.ServicePort,
				}},
				Selector: map[string]string{
					"some-label": "test",
				},
			},
		}

		_, err := fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create a Deployment with labels that match the service selector
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"some-label": "test",
						"version":    "v1",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"some-label": "test",
							"version":    "v1",
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "test-service-account",
						Containers: []corev1.Container{
							{
								Name:  "BookbuyerContainerA",
								Image: "random",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 80,
									},
								},
							},
						},
					},
				},
			},
		}

		_, err = fakeClientSet.AppsV1().Deployments(testNamespace).Create(context.Background(), deployment, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-provider.GetAnnouncementsChannel()

		givenSvcAccount := service.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account", // Should match the service account in the Deployment spec above
		}

		// Expect a MeshService that corresponds to a Service that matches the Deployment spec labels
		expectedMeshSvc := utils.K8sSvcToMeshSvc(svc)

		meshSvcs, err := provider.GetServicesForServiceAccount(givenSvcAccount)
		Expect(err).ToNot(HaveOccurred())
		expectedMeshSvcs := []service.MeshService{expectedMeshSvc}
		Expect(meshSvcs).To(Equal(expectedMeshSvcs))

		err = fakeClientSet.AppsV1().Deployments(testNamespace).Delete(context.Background(), deployment.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-provider.GetAnnouncementsChannel()
	})

	It("should return an error when the Service selector doesn't match the deployment", func() {
		// Create a Service
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: testNamespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Name:     "servicePort",
					Protocol: corev1.ProtocolTCP,
					Port:     tests.ServicePort,
				}},
				Selector: map[string]string{
					"app":                         "test",
					"key-specified-in-deployment": "no", // Since this label is missing in the deployment, the selector match should fail
				},
			},
		}

		_, err := fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create a Deployment with labels that match the service selector
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test",
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "test-service-account",
						Containers: []corev1.Container{
							{
								Name:  "BookbuyerContainerA",
								Image: "random",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 80,
									},
								},
							},
						},
					},
				},
			},
		}

		_, err = fakeClientSet.AppsV1().Deployments(testNamespace).Create(context.Background(), deployment, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-provider.GetAnnouncementsChannel()

		givenSvcAccount := service.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account", // Should match the service account in the Deployment spec above
		}

		// Expect a MeshService that corresponds to a Service that matches the Deployment spec labels
		_, err = provider.GetServicesForServiceAccount(givenSvcAccount)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(errDidNotFindServiceForServiceAccount))

		err = fakeClientSet.AppsV1().Deployments(testNamespace).Delete(context.Background(), deployment.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-provider.GetAnnouncementsChannel()
	})

	It("should return all services when multiple services match the same deployment", func() {
		// This test is meant to ensure the
		// service selector logic works as expected when multiple services
		// have the same selector match.

		// Create a Service
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: testNamespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Name:     "servicePort",
					Protocol: corev1.ProtocolTCP,
					Port:     tests.ServicePort,
				}},
				Selector: map[string]string{
					"some-label": "test",
				},
			},
		}

		_, err := fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create a second service with the same selector as the first service
		svc2 := *svc
		svc2.Name = "test-2"
		_, err = fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), &svc2, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create a Deployment with labels that match the service selector
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"some-label": "test",
						"version":    "v1",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"some-label": "test",
							"version":    "v1",
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "test-service-account",
						Containers: []corev1.Container{
							{
								Name:  "BookbuyerContainerA",
								Image: "random",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 80,
									},
								},
							},
						},
					},
				},
			},
		}

		_, err = fakeClientSet.AppsV1().Deployments(testNamespace).Create(context.Background(), deployment, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-provider.GetAnnouncementsChannel()

		givenSvcAccount := service.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account", // Should match the service account in the Deployment spec above
		}

		meshServices, err := provider.GetServicesForServiceAccount(givenSvcAccount)
		Expect(err).ToNot(HaveOccurred())
		expectedServices := []service.MeshService{
			{Name: "test-1", Namespace: testNamespace},
			{Name: "test-2", Namespace: testNamespace},
		}
		Expect(len(meshServices)).To(Equal(len(expectedServices)))
		Expect(meshServices[0]).To(BeElementOf(expectedServices))
		Expect(meshServices[1]).To(BeElementOf(expectedServices))

		err = fakeClientSet.AppsV1().Deployments(testNamespace).Delete(context.Background(), deployment.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-provider.GetAnnouncementsChannel()
	})
})
