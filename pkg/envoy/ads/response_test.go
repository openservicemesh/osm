package ads

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	catalogFake "github.com/openservicemesh/osm/pkg/catalog/fake"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test ADS response functions", func() {
	defer GinkgoRecover()

	mockCtrl := gomock.NewController(GinkgoT())
	proxyUUID := uuid.New()
	proxySvcID := tests.BookstoreServiceIdentity

	proxyRegistry := registry.NewProxyRegistry()

	proxy := envoy.NewProxy(envoy.KindSidecar, proxyUUID, proxySvcID, nil, 1)

	Context("Proxy is valid", func() {
		Expect(proxy).ToNot((BeNil()))
	})

	provider := compute.NewMockInterface(mockCtrl)
	provider.EXPECT().IsMetricsEnabled(gomock.Any()).Return(true, nil).AnyTimes()
	provider.EXPECT().ListEgressPoliciesForServiceAccount(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetIngressBackendPolicyForService(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetUpstreamTrafficSettingByService(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetUpstreamTrafficSettingByNamespace(gomock.Any()).Return(nil).AnyTimes()

	mc := catalogFake.NewFakeMeshCatalog(provider)

	Context("Test sendAllResponses()", func() {

		certManager := tresorFake.NewFake(1 * time.Hour)
		kubectrlMock := k8s.NewMockController(mockCtrl)

		provider.EXPECT().GetMeshConfig().Return(v1alpha2.MeshConfig{
			Spec: v1alpha2.MeshConfigSpec{
				Observability: v1alpha2.ObservabilitySpec{
					EnableDebugServer: true,
				},
			},
		}).AnyTimes()
		provider.EXPECT().ListServiceIdentitiesForService(gomock.Any()).Return(nil).AnyTimes()
		provider.EXPECT().ListServicesForProxy(proxy).Return(nil, nil).AnyTimes()

		metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.ProxyResponseSendSuccessCount)

		It("returns Aggregated Discovery Service response", func() {
			s := NewADSServer(mc, proxyRegistry, true, tests.Namespace, certManager, kubectrlMock, nil)

			Expect(s).ToNot(BeNil())
			snapshot, err := s.snapshotCache.GetSnapshot(proxy.UUID.String())
			Expect(err).To(HaveOccurred())
			Expect(snapshot).To(BeNil())

			err = s.update(proxy)
			Expect(err).To(BeNil())

			snapshot, err = s.snapshotCache.GetSnapshot(proxy.UUID.String())
			Expect(err).ToNot(HaveOccurred())
			Expect(snapshot).ToNot(BeNil())

			Expect(snapshot.GetVersion(string(envoy.TypeCDS))).To(Equal("1"))
			Expect(snapshot.GetResources(string(envoy.TypeCDS))).ToNot(BeNil())

			Expect(snapshot.GetVersion(string(envoy.TypeEDS))).To(Equal("1"))
			Expect(snapshot.GetResources(string(envoy.TypeEDS))).ToNot(BeNil())

			Expect(snapshot.GetVersion(string(envoy.TypeLDS))).To(Equal("1"))
			Expect(snapshot.GetResources(string(envoy.TypeLDS))).ToNot(BeNil())

			Expect(snapshot.GetVersion(string(envoy.TypeRDS))).To(Equal("1"))
			Expect(snapshot.GetResources(string(envoy.TypeRDS))).ToNot(BeNil())

			Expect(snapshot.GetVersion(string(envoy.TypeSDS))).To(Equal("1"))
			Expect(snapshot.GetResources(string(envoy.TypeSDS))).ToNot(BeNil())

			// Expect 2 SDS certs:
			// 1. Proxy's own cert to present to peer during mTLS/TLS handshake
			// 2. mTLS validation cert when this proxy is an upstream
			Expect(len(snapshot.GetResources(string(envoy.TypeSDS)))).To(Equal(2))

			// proxyServiceCert := xds_auth.Secret{}
			sdsResources := snapshot.GetResources(string(envoy.TypeSDS))

			resource, ok := sdsResources[secrets.NameForMTLSInbound]
			Expect(ok).To(BeTrue())
			Expect(resource).ToNot(BeNil())

			resource, ok = sdsResources[secrets.NameForIdentity(proxySvcID)]
			Expect(ok).To(BeTrue())
			Expect(resource).ToNot(BeNil())
		})
	})
})
