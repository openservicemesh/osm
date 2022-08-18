package fake

import (
	"time"

	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"

	"github.com/openservicemesh/osm/pkg/compute"

	"github.com/openservicemesh/osm/pkg/catalog"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/policy"
	smiFake "github.com/openservicemesh/osm/pkg/smi/fake"
)

// NewFakeMeshCatalog creates a new struct implementing catalog.MeshCataloger interface used for testing.
func NewFakeMeshCatalog(provider compute.Interface) *catalog.MeshCatalog {
	mockCtrl := gomock.NewController(ginkgo.GinkgoT())
	mockPolicyController := policy.NewMockController(mockCtrl)

	meshSpec := smiFake.NewFakeMeshSpecClient()

	stop := make(<-chan struct{})

	certManager := tresorFake.NewFake(1 * time.Hour)

	mockPolicyController.EXPECT().ListEgressPoliciesForSourceIdentity(gomock.Any()).Return(nil).AnyTimes()
	mockPolicyController.EXPECT().GetIngressBackendPolicy(gomock.Any()).Return(nil).AnyTimes()
	mockPolicyController.EXPECT().GetUpstreamTrafficSetting(gomock.Any()).Return(nil).AnyTimes()

	return catalog.NewMeshCatalog(meshSpec, certManager,
		mockPolicyController, stop, provider, messaging.NewBroker(stop))
}
