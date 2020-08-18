package kube

import (
	"context"
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test Kube Client Provider", func() {
	fakeClientSet := fake.NewSimpleClientset()
	stopChann := make(chan struct{})
	nsCtrl := namespace.NewFakeNamespaceController([]string{tests.BookbuyerService.Namespace})
	cfg := configurator.NewFakeConfigurator()
	providerID := "provider"

	cli, err := NewProvider(fakeClientSet, nsCtrl, stopChann, providerID, cfg)

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
		It("Testing GetServiceForServiceAccount", func() {
			c := NewFakeProvider()

			sMesh, err := c.GetServiceForServiceAccount(tests.BookstoreServiceAccount)
			Expect(err).To(BeNil())
			Expect(sMesh).To(Equal(tests.BookstoreService))

			sMesh2, err := c.GetServiceForServiceAccount(tests.BookbuyerServiceAccount)
			Expect(err).To(BeNil())
			Expect(sMesh2).To(Equal(tests.BookbuyerService))
		})
	})
})

var _ = Describe("When getting a Service associated with a ServiceAccount", func() {
	var (
		fakeClientSet *fake.Clientset
		provider      endpoint.Provider
		err           error
	)

	providerID := "test-provider"
	testNamespace := "test"
	nsCtrl := namespace.NewFakeNamespaceController([]string{testNamespace})
	cfg := configurator.NewFakeConfigurator()

	stop := make(chan struct{})

	BeforeEach(func() {
		fakeClientSet = fake.NewSimpleClientset()
		provider, err = NewProvider(fakeClientSet, nsCtrl, stop, providerID, cfg)
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
			_, err := provider.GetServiceForServiceAccount(tests.BookbuyerServiceAccount)
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
		expectedMeshSvc := service.MeshService{
			Namespace: svc.Namespace,
			Name:      svc.Name,
		}

		meshSvc, err := provider.GetServiceForServiceAccount(givenSvcAccount)
		Expect(err).ToNot(HaveOccurred())
		Expect(meshSvc).To(Equal(expectedMeshSvc))

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
		_, err = provider.GetServiceForServiceAccount(givenSvcAccount)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(errDidNotFindServiceForServiceAccount))

		err = fakeClientSet.AppsV1().Deployments(testNamespace).Delete(context.Background(), deployment.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-provider.GetAnnouncementsChannel()
	})

	It("should error when multiple services match the same deployment", func() {
		// This is a current limitation in OSM, this test is meant to ensure the
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

		_, err = provider.GetServiceForServiceAccount(givenSvcAccount)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(errMoreThanServiceForServiceAccount))

		err = fakeClientSet.AppsV1().Deployments(testNamespace).Delete(context.Background(), deployment.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-provider.GetAnnouncementsChannel()
	})
})
