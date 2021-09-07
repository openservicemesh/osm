package catalog

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestListServiceIdentitiesForService(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockServiceProvider := service.NewMockProvider(mockCtrl)
	mc := &MeshCatalog{
		serviceProviders: []service.Provider{mockServiceProvider},
	}

	testCases := []struct {
		svc                 service.MeshService
		expectedSvcAccounts []identity.ServiceIdentity
		expectedError       error
	}{
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			[]identity.ServiceIdentity{
				identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),
				identity.K8sServiceAccount{Name: "sa-2", Namespace: "ns-1"}.ToServiceIdentity(),
			},
			nil,
		},
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			[]identity.ServiceIdentity{
				identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),
				identity.K8sServiceAccount{Name: "sa-2", Namespace: "ns-1"}.ToServiceIdentity(),
			},
			nil,
		},
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			nil,
			errServiceNotFound,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			mockServiceProvider.EXPECT().ListServiceIdentitiesForService(tc.svc).Return(tc.expectedSvcAccounts, tc.expectedError).Times(1)
			serviceIdentities, err := mc.ListServiceIdentitiesForService(tc.svc)
			assert.ElementsMatch(serviceIdentities, tc.expectedSvcAccounts)
			assert.Equal(err, tc.expectedError)
		})
	}
}

func TestListMeshServices(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockServiceProvider := service.NewMockProvider(mockCtrl)
	mockKubeController := k8s.NewMockController(mockCtrl)
	mc := MeshCatalog{
		kubeController:   mockKubeController,
		serviceProviders: []service.Provider{mockServiceProvider},
	}

	testCases := []struct {
		name     string
		services map[string]string // name: namespace
	}{
		{
			name:     "services exist in mesh",
			services: map[string]string{"bookstore": "bookstore-ns", "bookbuyer": "bookbuyer-ns", "bookwarehouse": "bookwarehouse"},
		},
		{
			name:     "no services in mesh",
			services: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var expectedMeshServices, actual []service.MeshService

			for name, namespace := range tc.services {
				expectedMeshServices = append(expectedMeshServices, tests.NewMeshServiceFixture(name, namespace))
			}

			mockServiceProvider.EXPECT().ListServices().Return(expectedMeshServices, nil)
			for _, provider := range mc.serviceProviders {
				services, err := provider.ListServices()
				if err != nil {
					panic(err)
				}
				actual = append(actual, services...)
			}
			assert.Equal(expectedMeshServices, actual)
		})
	}
}
