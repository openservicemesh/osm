package catalog

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	specs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/service"

	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/tests"
)

type testParams struct {
	permissiveMode bool
}

func newFakeMeshCatalogForRoutes(t *testing.T, testParams testParams) *MeshCatalog {
	mockCtrl := gomock.NewController(t)

	stop := make(chan struct{})

	provider := compute.NewMockInterface(mockCtrl)
	provider.EXPECT().ListServices().Return([]service.MeshService{
		tests.BookstoreApexService,
		tests.BookbuyerService,
		tests.BookstoreV1Service,
		tests.BookstoreV2Service,
	}).AnyTimes()
	provider.EXPECT().GetServicesForServiceIdentity(gomock.Any()).Return([]service.MeshService{
		tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService,
	}).AnyTimes()
	provider.EXPECT().GetMeshConfig().Return(v1alpha2.MeshConfig{
		Spec: v1alpha2.MeshConfigSpec{
			Traffic: v1alpha2.TrafficSpec{
				EnablePermissiveTrafficPolicyMode: testParams.permissiveMode,
			},
		},
	}).AnyTimes()

	provider.EXPECT().ListTrafficTargets().Return([]*access.TrafficTarget{&tests.TrafficTarget, &tests.BookstoreV2TrafficTarget}).AnyTimes()
	provider.EXPECT().ListHTTPTrafficSpecs().Return([]*specs.HTTPRouteGroup{&tests.HTTPRouteGroup}).AnyTimes()
	provider.EXPECT().ListTrafficSplits().Return([]*split.TrafficSplit{}).AnyTimes()

	return NewMeshCatalog(provider, tresorFake.NewFake(1*time.Hour),
		stop, messaging.NewBroker(stop))
}
