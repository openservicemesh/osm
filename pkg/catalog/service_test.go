package catalog

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

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
