package kube

import (
	"fmt"
	"net"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/config"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test Multicluster functions of the Kubernetes endpoint", func() {
	defer GinkgoRecover()

	var client *Client

	mockCtrl := gomock.NewController(GinkgoT())
	mockKubeController := k8s.NewMockController(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigController := config.NewMockController(mockCtrl)

	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()

	toReturn := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Namespace: tests.BookbuyerService.Namespace},
		Subsets: []corev1.EndpointSubset{{
			Addresses: []corev1.EndpointAddress{{
				IP: "8.8.8.8",
			}},
			Ports: []corev1.EndpointPort{{
				Port: 88,
			}},
		}},
	}
	mockKubeController.EXPECT().GetEndpoints(tests.BookbuyerService).Return(toReturn, nil).AnyTimes()

	toReturnIdentities := []identity.K8sServiceAccount{tests.BookbuyerServiceAccount}
	mockKubeController.EXPECT().ListServiceIdentitiesForService(tests.BookbuyerService).Return(toReturnIdentities, nil).AnyTimes()

	expectedEndpoint := []endpoint.Endpoint{{
		IP:   net.IPv4(1, 2, 3, 4),
		Port: 5678,
	}}

	toReturnServices := []v1alpha1.MultiClusterService{{
		Spec: v1alpha1.MultiClusterServiceSpec{
			Clusters: []v1alpha1.ClusterSpec{{
				Address: fmt.Sprintf("%s:%d", expectedEndpoint[0].IP, expectedEndpoint[0].Port),
				Name:    "alpha",
			}},
			ServiceAccount: tests.BookbuyerServiceAccountName,
			Ports:          nil,
		},
	}}
	mockConfigController.EXPECT().GetMultiClusterServiceByServiceAccount(tests.BookbuyerServiceName, tests.Namespace).Return(toReturnServices).AnyTimes()

	BeforeEach(func() {
		client = NewClient(mockKubeController, mockConfigController, "kubernetes-endpoint-provider", mockConfigurator)
	})

	Context("Test getMulticlusterEndpoints()", func() {
		It("returns Multicluster endpoints for a service", func() {
			actual := client.getMulticlusterEndpoints(tests.BookbuyerService)
			Expect(actual).To(Equal(expectedEndpoint))
		})
	})

	Context("Test getMultiClusterServiceEndpointsForServiceAccount()", func() {
		It("returns Multicluster endpoints for a service account", func() {
			actual := client.getMultiClusterServiceEndpointsForServiceAccount(tests.BookbuyerServiceAccountName, tests.Namespace)
			Expect(actual).To(Equal(expectedEndpoint))
		})
	})

	Context("Test getIPPort()", func() {
		It("returns the port number specified in a ClusterSpec", func() {
			clusterSpec := v1alpha1.ClusterSpec{
				Address: "1.2.3.4:5678",
			}
			actualIP, actualPort, err := getIPPort(clusterSpec)
			Expect(err).ToNot(HaveOccurred())
			expectedIP := net.ParseIP("1.2.3.4")
			expectedPort := 5678
			Expect(actualIP).To(Equal(expectedIP))
			Expect(actualPort).To(Equal(expectedPort))
		})
	})
})
