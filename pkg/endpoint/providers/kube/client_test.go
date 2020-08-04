package kube

import (
	"context"
	"net"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test Kubernetes Provider", func() {
	Context("Testing ListServicesForServiceAccount", func() {
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
	})

	Context("Test getServicesFromLabelSelector()", func() {
		It("returns services from label selector", func() {
			kubeClient := testclient.NewSimpleClientset()
			namespaceController := namespace.NewFakeNamespaceController([]string{tests.Namespace})
			stop := make(chan struct{})
			cfg := configurator.NewFakeConfigurator()

			selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
			{
				// Create a matching Service
				svc := tests.NewServiceFixture(tests.BookstoreServiceName, tests.Namespace, selector)
				_, err := kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			{
				// Create another Service - does not match
				selector2 := map[string]string{tests.SelectorKey: uuid.New().String()}
				svc := tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, selector2)
				_, err := kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			provider, err := NewProvider(kubeClient, namespaceController, stop, constants.KubeProviderName, cfg)
			Expect(err).ToNot(HaveOccurred())

			actual, err := getServicesFromLabelSelector(*provider, selector, tests.Namespace)
			Expect(err).ToNot(HaveOccurred())

			Expect(actual.Cardinality()).To(Equal(1))
			actualService := (actual.ToSlice()[0]).(service.MeshService)
			Expect(actualService.Namespace).To(Equal(tests.Namespace))
			Expect(actualService.Name).To(Equal(tests.BookstoreServiceName))
		})
	})
})
