package service

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestServerName(t *testing.T) {
	assert := tassert.New(t)

	namespacedService := MeshService{
		Namespace: "namespace-here",
		Name:      "service-name-here",
	}
	assert.Equal("service-name-here.namespace-here.svc.cluster.local", namespacedService.ServerName())
}

func TestEquals(t *testing.T) {
	testCases := []struct {
		name           string
		service        MeshService
		anotherService MeshService
		isEqual        bool
	}{
		{
			name: "services are equal",
			service: MeshService{
				Namespace: "default",
				Name:      "bookbuyer",
			},
			anotherService: MeshService{
				Namespace: "default",
				Name:      "bookbuyer",
			},
			isEqual: true,
		},
		{
			name: "services are NOT equal",
			service: MeshService{
				Namespace: "default",
				Name:      "bookbuyer",
			},
			anotherService: MeshService{
				Namespace: "default",
				Name:      "bookstore",
			},
			isEqual: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := tc.service.Equals(tc.anotherService)
			assert.Equal(actual, tc.isEqual)
		})
	}
}

func TestString(t *testing.T) {
	testCases := []struct {
		name          string
		service       MeshService
		serviceString string
	}{
		{
			name: "service in default namespace",
			service: MeshService{
				Namespace: "default",
				Name:      "bookbuyer",
			},
			serviceString: "default/bookbuyer",
		},
		{
			name: "service in custom namespace",
			service: MeshService{
				Namespace: "bookbuyer-ns",
				Name:      "bookbuyer",
			},
			serviceString: "bookbuyer-ns/bookbuyer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := tc.service.String()
			assert.Equal(actual, tc.serviceString)
		})
	}
}
