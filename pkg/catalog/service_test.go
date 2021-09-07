package catalog

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
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
	}{
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			[]identity.ServiceIdentity{
				identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),
				identity.K8sServiceAccount{Name: "sa-2", Namespace: "ns-1"}.ToServiceIdentity(),
			},
		},
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			[]identity.ServiceIdentity{
				identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity(),
				identity.K8sServiceAccount{Name: "sa-2", Namespace: "ns-1"}.ToServiceIdentity(),
			},
		},
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			mockServiceProvider.EXPECT().ListServiceIdentitiesForService(tc.svc).Return(tc.expectedSvcAccounts).Times(1)
			serviceIdentities := mc.ListServiceIdentitiesForService(tc.svc)
			assert.ElementsMatch(serviceIdentities, tc.expectedSvcAccounts)
		})
	}
}

func TestListMeshServices(t *testing.T) {
	testCases := []struct {
		name                  string
		provider1MeshServices []service.MeshService
		provider2MeshServices []service.MeshService
		expectedMeshServices  []service.MeshService
	}{
		{
			name: "services exist in mesh",
			provider1MeshServices: []service.MeshService{
				{Namespace: "ns1", Name: "s1"},
				{Namespace: "ns2", Name: "s2"},
			},
			provider2MeshServices: []service.MeshService{
				{Namespace: "ns3", Name: "s3"},
				{Namespace: "ns4", Name: "s4"},
			},
			expectedMeshServices: []service.MeshService{
				{Namespace: "ns1", Name: "s1"},
				{Namespace: "ns2", Name: "s2"},
				{Namespace: "ns3", Name: "s3"},
				{Namespace: "ns4", Name: "s4"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			provider1 := service.NewMockProvider(mockCtrl)
			provider2 := service.NewMockProvider(mockCtrl)

			mc := MeshCatalog{
				serviceProviders: []service.Provider{provider1, provider2},
			}

			provider1.EXPECT().ListServices().Return(tc.provider1MeshServices)
			provider2.EXPECT().ListServices().Return(tc.provider2MeshServices)

			actual := mc.listMeshServices()
			assert.ElementsMatch(tc.expectedMeshServices, actual)
		})
	}
}
