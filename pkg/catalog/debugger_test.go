package catalog

import (
	"testing"

	"github.com/golang/mock/gomock"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestListSMIPolicies(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)

	splits := []*smiSplit.TrafficSplit{&tests.TrafficSplit}
	targets := []*smiAccess.TrafficTarget{&tests.TrafficTarget}
	httpRoutes := []*smiSpecs.HTTPRouteGroup{&tests.HTTPRouteGroup}
	svcAccounts := []identity.K8sServiceAccount{tests.BookbuyerServiceAccount}

	mockMeshSpec.EXPECT().ListTrafficSplits().Return(splits)
	mockMeshSpec.EXPECT().ListTrafficTargets().Return(targets)
	mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return(httpRoutes)
	mockMeshSpec.EXPECT().ListServiceAccounts().Return(svcAccounts)

	mc := &MeshCatalog{
		meshSpec: mockMeshSpec,
	}

	a := assert.New(t)

	trafficSplits, serviceAccounts, trafficSpecs, trafficTargets := mc.ListSMIPolicies()
	a.ElementsMatch(trafficSplits, splits)
	a.ElementsMatch(targets, trafficTargets)
	a.ElementsMatch(httpRoutes, trafficSpecs)
	a.ElementsMatch(svcAccounts, serviceAccounts)
}
