package service

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"
)

func TestUnmarshalMeshService(t *testing.T) {
	assert := tassert.New(t)
	require := trequire.New(t)

	namespace := "randomNamespace"
	serviceName := "randomServiceName"
	meshService := &MeshService{
		Namespace: namespace,
		Name:      serviceName,
	}
	str := meshService.String()
	fmt.Println(str)

	testCases := []struct {
		name          string
		expectedErr   bool
		serviceString string
	}{
		{
			name:          "successfully unmarshal service",
			expectedErr:   false,
			serviceString: "randomNamespace/randomServiceName",
		},
		{
			name:          "incomplete namespaced service name 1",
			expectedErr:   true,
			serviceString: "/svnc",
		},
		{
			name:          "incomplete namespaced service name 2",
			expectedErr:   true,
			serviceString: "svnc/",
		},
		{
			name:          "incomplete namespaced service name 3",
			expectedErr:   true,
			serviceString: "/svnc/",
		},
		{
			name:          "incomplete namespaced service name 3",
			expectedErr:   true,
			serviceString: "/",
		},
		{
			name:          "incomplete namespaced service name 3",
			expectedErr:   true,
			serviceString: "",
		},
		{
			name:          "incomplete namespaced service name 3",
			expectedErr:   true,
			serviceString: "test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := UnmarshalMeshService(tc.serviceString)
			if tc.expectedErr {
				assert.NotNil(err)
			} else {
				require.Nil(err)
				assert.Equal(meshService, actual)
			}
		})
	}
}

func TestServerName(t *testing.T) {
	assert := tassert.New(t)

	namespacedService := MeshService{
		Namespace: "namespace-here",
		Name:      "service-name-here",
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
			actual := tc.service.String()
			assert.Equal(actual, tc.serviceString)
		})
	}
}
