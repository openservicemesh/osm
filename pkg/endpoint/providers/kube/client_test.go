package kube

import (
	"context"
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	fake "k8s.io/client-go/kubernetes/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test Kube Client Provider", func() {
	Context("Testing ListServicesForServiceAccount", func() {
		f := fake.NewSimpleClientset()
		stop := make(chan struct{})
		namespaceController := namespace.NewFakeNamespaceController([]string{tests.BookbuyerService.Namespace})
		cfg := configurator.NewFakeConfigurator()
		provider := "provider"

		cli, err := NewProvider(f, namespaceController, stop, provider, cfg)

		It("Check global provider creation succeeded", func() {
			Expect(err).To(BeNil())
		})

		It("Test Provider APIs", func() {
			// New provider, no errs
			id := cli.GetID()
			Expect(id).To(Equal(provider))
		})

		It("Test ListEndpointsForService", func() {
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

			f.CoreV1().Endpoints(tests.BookbuyerService.Namespace).
				Create(context.TODO(), endp, metav1.CreateOptions{})

			// Expect bookbuyer endpoint, async/event listening in provider makes inmediate calls fail
			Eventually(func() []endpoint.Endpoint {
				return cli.ListEndpointsForService(tests.BookbuyerService)
			}, 2*time.Second).Should(Equal([]endpoint.Endpoint{
				{
					IP:   net.IPv4(8, 8, 8, 8),
					Port: 88,
				},
			}))
		})

		It("Test GetServiceForServiceAccount", func() {
			svcMesh, err := cli.GetServiceForServiceAccount(tests.BookbuyerServiceAccount)

			// Expect err to be thrown as the Service Account does not exist, empty results
			Expect(svcMesh).To(Equal(service.MeshService{}))
			Expect(err).ToNot(BeNil())

			// Add bookbuyer deployment
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tests.BookbuyerService.Name,
					Namespace: tests.BookbuyerService.Namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "bookbuyer",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "bookbuyer",
							},
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: tests.BookbuyerServiceAccountName,
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

			// Add deployment
			_, err = f.AppsV1().Deployments(tests.BookbuyerService.Namespace).
				Create(context.Background(), deployment, metav1.CreateOptions{})

			Expect(err).To(BeNil())

			// Expect bookbuyer endpoint, async/event listening in provider makes inmediate calls fail
			Eventually(func() service.MeshService {
				svcMesh, _ := cli.GetServiceForServiceAccount(tests.BookbuyerServiceAccount)
				return svcMesh
			}, 2*time.Second).Should(Equal(tests.BookbuyerService))

		})

		It("Test GetAnnouncementChannel", func() {
			ch := cli.GetAnnouncementsChannel()

			// Can only expect to not be null
			Expect(ch).ToNot(BeNil())
		})
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
