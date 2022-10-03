package server

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	catalogFake "github.com/openservicemesh/osm/pkg/catalog/fake"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/generator"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestADSResponse(t *testing.T) {
	a := assert.New(t)

	mockCtrl := gomock.NewController(t)
	proxyUUID := uuid.New()
	proxySvcID := tests.BookstoreServiceIdentity

	proxy := models.NewProxy(models.KindSidecar, proxyUUID, proxySvcID, nil, 1)
	a.NotNil(proxy)

	provider := compute.NewMockInterface(mockCtrl)
	provider.EXPECT().IsMetricsEnabled(gomock.Any()).Return(true, nil).AnyTimes()
	provider.EXPECT().ListEgressPoliciesForServiceAccount(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetIngressBackendPolicyForService(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetUpstreamTrafficSettingByService(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetUpstreamTrafficSettingByNamespace(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().ListTrafficTargets().Return(nil).AnyTimes()
	provider.EXPECT().GetTelemetryConfig(gomock.Any()).Return(models.TelemetryConfig{}).AnyTimes()

	mc := catalogFake.NewFakeMeshCatalog(provider)

	certManager := tresorFake.NewFake(1 * time.Hour)

	provider.EXPECT().GetMeshConfig().Return(v1alpha2.MeshConfig{
		Spec: v1alpha2.MeshConfigSpec{
			Observability: v1alpha2.ObservabilitySpec{
				EnableDebugServer: true,
			},
		},
	}).AnyTimes()
	provider.EXPECT().ListServiceIdentitiesForService(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	provider.EXPECT().ListServicesForProxy(proxy).Return(nil, nil).AnyTimes()

	metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.ProxyResponseSendSuccessCount)

	s := NewADSServer()
	ctx := context.Background()
	a.NotNil(s)
	snapshot, err := s.snapshotCache.GetSnapshot(proxy.UUID.String())
	a.NotNil(err)
	a.Nil(snapshot)

	g := generator.NewEnvoyConfigGenerator(mc, certManager)

	resources, err := g.GenerateConfig(ctx, proxy)
	a.Nil(err)

	err = s.UpdateProxy(ctx, proxy, resources)
	a.Nil(err)

	snapshot, err = s.snapshotCache.GetSnapshot(proxy.UUID.String())
	a.Nil(err)
	a.NotNil(snapshot)

	a.Equal(snapshot.GetVersion(string(envoy.TypeCDS)), "1")
	a.NotNil(snapshot.GetResources(string(envoy.TypeCDS)))

	a.Equal(snapshot.GetVersion(string(envoy.TypeEDS)), "1")
	a.NotNil(snapshot.GetResources(string(envoy.TypeEDS)))

	a.Equal(snapshot.GetVersion(string(envoy.TypeLDS)), "1")
	a.NotNil(snapshot.GetResources(string(envoy.TypeLDS)))

	a.Equal(snapshot.GetVersion(string(envoy.TypeRDS)), "1")
	a.NotNil(snapshot.GetResources(string(envoy.TypeRDS)))

	a.Equal(snapshot.GetVersion(string(envoy.TypeSDS)), "1")
	a.NotNil(snapshot.GetResources(string(envoy.TypeSDS)))

	// Expect 2 SDS certs:
	// 1. Proxy's own cert to present to peer during mTLS/TLS handshake
	// 2. mTLS validation cert when this proxy is an upstream
	a.Equal(len(snapshot.GetResources(string(envoy.TypeSDS))), 2)

	// proxyServiceCert := xds_auth.Secret{}
	sdsResources := snapshot.GetResources(string(envoy.TypeSDS))

	resource, ok := sdsResources[secrets.NameForMTLSInbound]
	a.True(ok)
	a.NotNil(resource)

	resource, ok = sdsResources[secrets.NameForIdentity(proxySvcID)]
	a.True(ok)
	a.NotNil(resource)
}
