package kube

import (
	"fmt"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/config"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestHelperFunctions(t *testing.T) {
	assert := tassert.New(t)

	var c *client

	mockCtrl := gomock.NewController(t)
	mockKubeController := k8s.NewMockController(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigController := config.NewMockController(mockCtrl)

	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()

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
		Zone: "alpha",
	}}

	toReturnServices := []configv1alpha3.MultiClusterService{{
		Spec: configv1alpha3.MultiClusterServiceSpec{
			Clusters: []configv1alpha3.ClusterSpec{{
				Address: fmt.Sprintf("%s:%d", expectedEndpoint[0].IP, expectedEndpoint[0].Port),
				Name:    "alpha",
			}},
			ServiceAccount: tests.BookbuyerServiceAccountName,
			Ports:          nil,
		},
	}}
	mockConfigController.EXPECT().GetMultiClusterServiceByServiceAccount(tests.BookbuyerServiceName, tests.Namespace).Return(toReturnServices).AnyTimes()

	c = NewClient(mockKubeController, mockConfigController, mockConfigurator)

	// Test getMulticlusterEndpoints()
	// returns Multicluster endpoints for a service
	actual := c.getMulticlusterEndpoints(tests.BookbuyerService)
	assert.Equal(actual, expectedEndpoint)

	// Test getMultiClusterServiceEndpointsForServiceAccount()
	// returns Multicluster endpoints for a service account
	actual = c.getMultiClusterServiceEndpointsForServiceAccount(tests.BookbuyerServiceAccountName, tests.Namespace)
	assert.Equal(actual, expectedEndpoint)

	// Test getIPPort()
	// returns the port number specified in a ClusterSpec
	clusterSpec := configv1alpha3.ClusterSpec{
		Address: "1.2.3.4:5678",
	}
	actualIP, actualPort, err := getIPPort(clusterSpec)
	assert.Equal(err, nil)
	expectedIP := net.ParseIP("1.2.3.4")
	expectedPort := 5678
	assert.Equal(actualIP, expectedIP)
	assert.Equal(actualPort, expectedPort)
}
