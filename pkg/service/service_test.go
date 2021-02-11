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

func TestGetSyntheticService(t *testing.T) {
	assert := tassert.New(t)

	namespace := "-namespace-"
	serviceAccount := "-service-account-"

	sa := K8sServiceAccount{
		Namespace: namespace,
		Name:      serviceAccount,
	}
	expected := MeshService{
		Namespace: namespace,
		Name:      fmt.Sprintf("-service-account-.-namespace-.osm.synthetic-%s", SyntheticServiceSuffix),
	}

	actual := sa.GetSyntheticService()

	assert.Equal(expected, actual)
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

func TestIsSyntheticService(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name        string
		meshService MeshService
		expected    bool
	}{
		{
			name: "is not synthetic",
			meshService: MeshService{
				Namespace: "namespace-here",
				Name:      "service-name-here",
			},
			expected: false,
		},
		{
			name: "is synthetic",
			meshService: MeshService{
				Namespace: "namespace",
				Name:      fmt.Sprintf("-service-account-.-namespace-.osm.synthetic-%s", SyntheticServiceSuffix),
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.meshService.IsSyntheticService()
			assert.Equal(tc.expected, actual)
		})
	}
}
