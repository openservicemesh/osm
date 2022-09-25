package generator

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/catalog"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGenerateConfig(t *testing.T) {
	tassert := assert.New(t)
	proxyUUID := uuid.New()
	proxy := models.NewProxy(models.KindSidecar, proxyUUID, tests.BookbuyerServiceIdentity, nil, 1)

	mockCtrl := gomock.NewController(t)

	provider := compute.NewMockInterface(mockCtrl)
	provider.EXPECT().IsMetricsEnabled(gomock.Any()).Return(true, nil).AnyTimes()
	provider.EXPECT().ListEgressPoliciesForServiceAccount(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetIngressBackendPolicyForService(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetUpstreamTrafficSettingByService(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetUpstreamTrafficSettingByNamespace(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetMeshConfig().Return(configv1alpha2.MeshConfig{
		Spec: configv1alpha2.MeshConfigSpec{
			Traffic: configv1alpha2.TrafficSpec{
				EnablePermissiveTrafficPolicyMode: true,
			},
		},
	}).AnyTimes()
	provider.EXPECT().ListServiceIdentitiesForService(gomock.Any(), gomock.Any()).Return([]identity.ServiceIdentity{tests.BookstoreServiceIdentity}, nil).AnyTimes()
	provider.EXPECT().ListServices().Return([]service.MeshService{tests.BookstoreApexService, tests.BookbuyerService}).AnyTimes()
	provider.EXPECT().ListServicesForProxy(proxy).Return([]service.MeshService{tests.BookbuyerService}, nil).AnyTimes()
	provider.EXPECT().GetHostnamesForService(gomock.Any(), gomock.Any()).Return([]string{"fake.hostname.cluster.local"}).AnyTimes()
	provider.EXPECT().GetServicesForServiceIdentity(tests.BookstoreServiceIdentity).Return([]service.MeshService{tests.BookstoreApexService}).AnyTimes()
	provider.EXPECT().GetServicesForServiceIdentity(tests.BookbuyerServiceIdentity).Return([]service.MeshService{tests.BookbuyerService}).AnyTimes()
	provider.EXPECT().GetResolvableEndpointsForService(tests.BookbuyerService).Return([]endpoint.Endpoint{
		{
			IP:   net.IPv4(8, 8, 8, 8),
			Port: endpoint.Port(8080),
		},
	}).AnyTimes()
	provider.EXPECT().GetResolvableEndpointsForService(tests.BookstoreApexService).Return([]endpoint.Endpoint{
		{
			IP:   net.IPv4(8, 0, 8, 0),
			Port: endpoint.Port(8080),
		},
	}).AnyTimes()
	provider.EXPECT().ListEndpointsForService(tests.BookbuyerService).Return([]endpoint.Endpoint{
		{
			IP:   net.IPv4(10, 12, 10, 12),
			Port: endpoint.Port(8080),
		},
	}).AnyTimes()
	provider.EXPECT().ListEndpointsForService(tests.BookstoreApexService).Return([]endpoint.Endpoint{
		{
			IP:   net.IPv4(10, 10, 10, 10),
			Port: endpoint.Port(8080),
		},
	}).AnyTimes()
	provider.EXPECT().ListTrafficSplits().Return(nil).AnyTimes()
	provider.EXPECT().GetTelemetryConfig(gomock.Any()).Return(models.TelemetryConfig{}).AnyTimes()

	certManager := tresorFake.NewFake(time.Hour)
	mc := catalog.NewMeshCatalog(provider, certManager)

	g := NewEnvoyConfigGenerator(mc, certManager)

	resources, err := g.GenerateConfig(context.Background(), proxy)

	tassert.Len(resources, 5)
	tassert.NoError(err)

	for typ, resource := range resources {
		tassert.Greater(len(resource), 0, fmt.Sprintf("resource type %s is empty", typ))
	}

	snapshot, err := cache.NewSnapshot("1", resources)
	tassert.NoError(err)

	err = snapshot.Consistent()
	tassert.NoError(err)
}
