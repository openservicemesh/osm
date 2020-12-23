package catalog

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestListServiceAccountsForService(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := kubernetes.NewMockController(mockCtrl)
	mc := &MeshCatalog{
		kubeController: mockKubeController,
	}

	testCases := []struct {
		svc                 service.MeshService
		expectedSvcAccounts []service.K8sServiceAccount
		expectedError       error
	}{
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			[]service.K8sServiceAccount{{Name: "sa-1", Namespace: "ns-1"}, {Name: "sa-2", Namespace: "ns-1"}},
			nil,
		},
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			[]service.K8sServiceAccount{{Name: "sa-1", Namespace: "ns-1"}, {Name: "sa-2", Namespace: "ns-1"}},
			nil,
		},
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			nil,
			errors.New("test error"),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			mockKubeController.EXPECT().ListServiceAccountsForService(tc.svc).Return(tc.expectedSvcAccounts, tc.expectedError).Times(1)

			svcAccounts, err := mc.ListServiceAccountsForService(tc.svc)
			assert.ElementsMatch(svcAccounts, tc.expectedSvcAccounts)
			assert.Equal(err, tc.expectedError)
		})
	}
}

func TestGetPortToProtocolMappingForService(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type endpointProviderConfig struct {
		provider          *endpoint.MockProvider
		portToProtocolMap map[uint32]string
		err               error
	}

	testCases := []struct {
		name                      string
		providerConfigs           []endpointProviderConfig
		expectedPortToProtocolMap map[uint32]string
		expectError               bool
	}{
		{
			// Test case 1
			name: "multiple providers correctly returning the same port:protocol mapping",
			providerConfigs: []endpointProviderConfig{
				{
					// provider 1
					provider:          endpoint.NewMockProvider(mockCtrl),
					portToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
					err:               nil,
				},
				{
					// provider 2
					provider:          endpoint.NewMockProvider(mockCtrl),
					portToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
					err:               nil,
				},
			},
			expectedPortToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
			expectError:               false,
		},

		{
			// Test case 2
			name: "multiple providers incorrectly returning different port:protocol mapping",
			providerConfigs: []endpointProviderConfig{
				{
					// provider 1
					provider:          endpoint.NewMockProvider(mockCtrl),
					portToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
					err:               nil,
				},
				{
					// provider 2
					provider:          endpoint.NewMockProvider(mockCtrl),
					portToProtocolMap: map[uint32]string{80: "tcp", 90: "http"},
					err:               nil,
				},
			},
			expectedPortToProtocolMap: nil,
			expectError:               true,
		},

		{
			// Test case 3
			name: "single provider correctly returning port:protocol mapping",
			providerConfigs: []endpointProviderConfig{
				{
					// provider 1
					provider:          endpoint.NewMockProvider(mockCtrl),
					portToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
					err:               nil,
				},
			},
			expectedPortToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
			expectError:               false,
		},
	}

	testSvc := service.MeshService{Name: "foo", Namespace: "bar"}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			// Create a list of providers for catalog and mock their calls based on the given config
			var allProviders []endpoint.Provider
			for _, providerConfig := range tc.providerConfigs {
				allProviders = append(allProviders, providerConfig.provider)
				providerConfig.provider.EXPECT().GetPortToProtocolMappingForService(testSvc).Return(providerConfig.portToProtocolMap, providerConfig.err).Times(1)
			}

			mc := &MeshCatalog{
				endpointsProviders: allProviders,
			}

			actualPortToProtocolMap, err := mc.GetPortToProtocolMappingForService(testSvc)

			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedPortToProtocolMap, actualPortToProtocolMap)
		})
	}
}
