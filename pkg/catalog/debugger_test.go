package catalog

import (
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/tests"
)

func TestListSMIPolicies(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockCatalog := newFakeMeshCatalog()

	trafficSplits, serviceAccounts, routeGroups, trafficTargets := mockCatalog.ListSMIPolicies()
	assert.Equal(trafficSplits[0].Spec.Service, "bookstore-apex")
	assert.Equal(serviceAccounts[0].String(), "default/bookstore")
	assert.Equal(routeGroups[0].Name, "bookstore-service-routes")
	assert.Equal(trafficTargets[0].Name, tests.TrafficTargetName)
}
