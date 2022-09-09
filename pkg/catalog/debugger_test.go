package catalog

import (
	"testing"

	"github.com/golang/mock/gomock"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestListSMIPolicies(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCompute := compute.NewMockInterface(mockCtrl)

	splits := []*smiSplit.TrafficSplit{&tests.TrafficSplit}
	targets := []*smiAccess.TrafficTarget{&tests.TrafficTarget}
	httpRoutes := []*smiSpecs.HTTPRouteGroup{&tests.HTTPRouteGroup}
	svcAccounts := []identity.K8sServiceAccount{tests.BookbuyerServiceAccount}

	mockCompute.EXPECT().ListTrafficSplits().Return(splits)
	mockCompute.EXPECT().ListTrafficTargets().Return(targets)
	mockCompute.EXPECT().ListHTTPTrafficSpecs().Return(httpRoutes)
	mockCompute.EXPECT().ListServiceAccountsFromTrafficTargets().Return(svcAccounts)

	mc := &MeshCatalog{
		Interface: mockCompute,
	}

	a := assert.New(t)

	trafficSplits, serviceAccounts, trafficSpecs, trafficTargets := mc.ListSMIPolicies()
	a.ElementsMatch(trafficSplits, splits)
	a.ElementsMatch(targets, trafficTargets)
	a.ElementsMatch(httpRoutes, trafficSpecs)
	a.ElementsMatch(svcAccounts, serviceAccounts)
}
