package service

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestServerName(t *testing.T) {
	assert := tassert.New(t)

	namespacedService := MeshService{
		Namespace:     "namespace-here",
		Name:          "service-name-here",
		ClusterDomain: constants.ClusterDomain,
	}
	actual := namespacedService.ServerName()
	assert.Equal("service-name-here.namespace-here.svc.cluster.local", actual)
}

func TestEquals(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name           string
		service        MeshService
		anotherService MeshService
		isEqual        bool
	}{
		{
			name: "services are equal",
			service: MeshService{
				Namespace:     "default",
				Name:          "bookbuyer",
				ClusterDomain: constants.ClusterDomain,
			},
			anotherService: MeshService{
				Namespace:     "default",
				Name:          "bookbuyer",
				ClusterDomain: constants.ClusterDomain,
			},
			isEqual: true,
		},
		{
			name: "services are NOT equal",
			service: MeshService{
				Namespace:     "default",
				Name:          "bookbuyer",
				ClusterDomain: constants.ClusterDomain,
			},
			anotherService: MeshService{
				Namespace:     "default",
				Name:          "bookstore",
				ClusterDomain: constants.ClusterDomain,
			},
			isEqual: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.service.Equals(tc.anotherService)
			assert.Equal(actual, tc.isEqual)
		})
	}
}

func TestString(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name          string
		service       MeshService
		serviceString string
	}{
		{
			name: "service in default namespace",
			service: MeshService{
				Namespace:     "default",
				Name:          "bookbuyer",
				ClusterDomain: constants.ClusterDomain,
			},
			serviceString: "default/bookbuyer/local",
		},
		{
			name: "service in custom namespace",
			service: MeshService{
				Namespace:     "bookbuyer-ns",
				Name:          "bookbuyer",
				ClusterDomain: constants.ClusterDomain,
			},
			serviceString: "bookbuyer-ns/bookbuyer/local",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.service.String()
			assert.Equal(actual, tc.serviceString)
		})
	}
}
