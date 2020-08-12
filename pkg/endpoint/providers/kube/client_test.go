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
	Context("Test kube endpoint provider APIs", func() {
		fakeClientSet := fake.NewSimpleClientset()
		stopChann := make(chan struct{})
		nsCtrl := namespace.NewFakeNamespaceController([]string{tests.BookbuyerService.Namespace})
		cfg := configurator.NewFakeConfigurator()
		providerID := "provider"

		cli, err := NewProvider(fakeClientSet, nsCtrl, stopChann, providerID, cfg)

		It("verifies new Provider at context scope succeeded", func() {
			Expect(err).To(BeNil())
		})

		It("tests GetID", func() {
			Expect(cli.GetID()).To(Equal(providerID))
		})

		It("tests ListEndpointsForService", func() {
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

			// Expect bookbuyer endpoint, async/event listening in Provider makes inmediate calls fail
			Eventually(func() []endpoint.Endpoint {
				return cli.ListEndpointsForService(tests.BookbuyerService)
			}, 2*time.Second).Should(Equal([]endpoint.Endpoint{
				{
					IP:   net.IPv4(8, 8, 8, 8),
					Port: 88,
				},
			}))
		})

		It("tests GetServiceForServiceAccount", func() {
			// No deployments set, this returns err & empty svcMesh
			svcMesh, err := cli.GetServiceForServiceAccount(tests.BookbuyerServiceAccount)
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
							"app": tests.BookbuyerService.Name,
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": tests.BookbuyerService.Name,
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
			_, err = fakeClientSet.AppsV1().Deployments(tests.BookbuyerService.Namespace).
				Create(context.Background(), deployment, metav1.CreateOptions{})

			Expect(err).To(BeNil())

			// Expect bookbuyer endpoint, async/event listening in provider makes inmediate calls fail
			Eventually(func() service.MeshService {
				svcMesh, _ := cli.GetServiceForServiceAccount(tests.BookbuyerServiceAccount)
				return svcMesh
			}, 2*time.Second).Should(Equal(tests.BookbuyerService))

		})

		It("tests GetAnnouncementChannel", func() {
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
