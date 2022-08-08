package ads

import (
	"fmt"
	"time"

	mapset "github.com/deckarep/golang-set"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	catalogFake "github.com/openservicemesh/osm/pkg/catalog/fake"
	"github.com/openservicemesh/osm/pkg/certificate"
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
	fakeCertManager, err := certificate.FakeCertManager()
	Expect(err).ToNot(HaveOccurred())
	proxyUUID := uuid.New()
	proxySvcAccount := tests.BookstoreServiceAccount

	proxyRegistry := registry.NewProxyRegistry()

	proxy := envoy.NewProxy(envoy.KindSidecar, proxyUUID, proxySvcAccount.ToServiceIdentity(), nil, 1)

	Context("Proxy is valid", func() {
		Expect(proxy).ToNot((BeNil()))
	})

	provider := compute.NewMockInterface(mockCtrl)
	provider.EXPECT().IsMetricsEnabled(gomock.Any()).Return(true, nil).AnyTimes()

	mc := catalogFake.NewFakeMeshCatalog(provider)

	Context("Test sendAllResponses()", func() {

		certManager := tresorFake.NewFake(1 * time.Hour)
		certPEM, _ := certManager.IssueCertificate(proxySvcAccount.ToServiceIdentity().String(), certificate.Service)
		cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())
		server, actualResponses := tests.NewFakeXDSServer(cert, nil, nil)
		kubectrlMock := k8s.NewMockController(mockCtrl)

		provider.EXPECT().GetMeshConfig().Return(v1alpha2.MeshConfig{
			Spec: v1alpha2.MeshConfigSpec{
				Observability: v1alpha2.ObservabilitySpec{
					EnableDebugServer: true,
				},
			},
		}).AnyTimes()
		provider.EXPECT().ListServiceIdentitiesForService(gomock.Any()).Return(nil).AnyTimes()
		provider.EXPECT().GetServicesForProxy(proxy).Return(nil, nil).AnyTimes()

		metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.ProxyResponseSendSuccessCount)

		It("returns Aggregated Discovery Service response", func() {
			s := NewADSServer(mc, proxyRegistry, true, tests.Namespace, fakeCertManager, kubectrlMock, nil)

			Expect(s).ToNot(BeNil())

			// Set subscribed resources for SDS
			proxy.SetSubscribedResources(envoy.TypeSDS, mapset.NewSetWith("service-cert:default/bookstore", "root-cert-for-mtls-inbound:default/bookstore"))

			err := s.sendResponse(proxy, &server, nil, envoy.XDSResponseOrder...)
			Expect(err).To(BeNil())
			Expect(actualResponses).ToNot(BeNil())
			Expect(len(*actualResponses)).To(Equal(5))

			Expect((*actualResponses)[0].VersionInfo).To(Equal("1"))
			Expect((*actualResponses)[0].TypeUrl).To(Equal(string(envoy.TypeCDS)))

			Expect((*actualResponses)[1].VersionInfo).To(Equal("1"))
			Expect((*actualResponses)[1].TypeUrl).To(Equal(string(envoy.TypeEDS)))

			Expect((*actualResponses)[2].VersionInfo).To(Equal("1"))
			Expect((*actualResponses)[2].TypeUrl).To(Equal(string(envoy.TypeLDS)))

			Expect((*actualResponses)[3].VersionInfo).To(Equal("1"))
			Expect((*actualResponses)[3].TypeUrl).To(Equal(string(envoy.TypeRDS)))

			Expect((*actualResponses)[4].VersionInfo).To(Equal("1"))
			Expect((*actualResponses)[4].TypeUrl).To(Equal(string(envoy.TypeSDS)))
			log.Printf("%v", len((*actualResponses)[4].Resources))

			// Expect 3 SDS certs:
			// 1. Proxy's own cert to present to peer during mTLS/TLS handshake
			// 2. mTLS validation cert when this proxy is an upstream
			Expect(len((*actualResponses)[4].Resources)).To(Equal(2))

			var tmpResource *any.Any

			proxyServiceCert := xds_auth.Secret{}
			tmpResource = (*actualResponses)[4].Resources[0]
			err = tmpResource.UnmarshalTo(&proxyServiceCert)
			Expect(err).To(BeNil())
			Expect(proxyServiceCert.Name).To(Equal(secrets.SDSCert{
				Name:     proxySvcAccount.String(),
				CertType: secrets.RootCertTypeForMTLSInbound,
			}.String()))

			serverRootCertTypeForMTLSInbound := xds_auth.Secret{}
			tmpResource = (*actualResponses)[4].Resources[1]
			err = tmpResource.UnmarshalTo(&serverRootCertTypeForMTLSInbound)
			Expect(err).To(BeNil())
			Expect(serverRootCertTypeForMTLSInbound.Name).To(Equal(secrets.SDSCert{
				Name:     proxySvcAccount.String(),
				CertType: secrets.ServiceCertType,
			}.String()))

			Expect(metricsstore.DefaultMetricsStore.Contains(fmt.Sprintf("osm_proxy_response_send_success_count{identity=%q,proxy_uuid=%q,type=%q} 1\n", proxy.Identity, proxy.UUID, envoy.TypeCDS))).To(BeTrue())
		})
	})

	Context("Test sendSDSResponse()", func() {

		certManager := tresorFake.NewFake(1 * time.Hour)
		certCNPrefix := fmt.Sprintf("%s.%s.%s.%s", uuid.New(), envoy.KindSidecar, proxySvcAccount.Name, proxySvcAccount.Namespace)
		certPEM, _ := certManager.IssueCertificate(certCNPrefix, certificate.Service)
		cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())
		server, actualResponses := tests.NewFakeXDSServer(cert, nil, nil)
		kubectrlMock := k8s.NewMockController(mockCtrl)

		provider.EXPECT().GetMeshConfig().Return(v1alpha2.MeshConfig{
			Spec: v1alpha2.MeshConfigSpec{
				Observability: v1alpha2.ObservabilitySpec{
					EnableDebugServer: true,
				},
			},
		}).AnyTimes()
		provider.EXPECT().ListServiceIdentitiesForService(gomock.Any()).Return(nil).AnyTimes()

		It("returns Aggregated Discovery Service response", func() {
			s := NewADSServer(mc, proxyRegistry, true, tests.Namespace, fakeCertManager, kubectrlMock, nil)

			Expect(s).ToNot(BeNil())

			// Set subscribed resources
			proxy.SetSubscribedResources(envoy.TypeSDS, mapset.NewSetWith("service-cert:default/bookstore", "root-cert-for-mtls-inbound:default/bookstore"))

			err := s.sendResponse(proxy, &server, nil, envoy.TypeSDS)
			Expect(err).To(BeNil())
			Expect(actualResponses).ToNot(BeNil())
			Expect(len(*actualResponses)).To(Equal(1))

			sdsResponse := (*actualResponses)[0]

			Expect(sdsResponse.VersionInfo).To(Equal("2")) // 2 because first update was by the previous test for the proxy
			Expect(sdsResponse.TypeUrl).To(Equal(string(envoy.TypeSDS)))

			// Expect 3 SDS certs:
			// 1. Proxy's own cert to present to peer during mTLS/TLS handshake
			// 2. mTLS validation cert when this proxy is an upstream
			Expect(len(sdsResponse.Resources)).To(Equal(2))

			var tmpResource *any.Any

			proxyServiceCert := xds_auth.Secret{}
			tmpResource = sdsResponse.Resources[0]
			err = tmpResource.UnmarshalTo(&proxyServiceCert)
			Expect(err).To(BeNil())
			Expect(proxyServiceCert.Name).To(Equal(secrets.SDSCert{
				Name:     proxySvcAccount.String(),
				CertType: secrets.RootCertTypeForMTLSInbound,
			}.String()))

			serverRootCertTypeForMTLSInbound := xds_auth.Secret{}
			tmpResource = sdsResponse.Resources[1]
			err = tmpResource.UnmarshalTo(&serverRootCertTypeForMTLSInbound)
			Expect(err).To(BeNil())
			Expect(serverRootCertTypeForMTLSInbound.Name).To(Equal(secrets.SDSCert{
				Name:     proxySvcAccount.String(),
				CertType: secrets.ServiceCertType,
			}.String()))
		})
	})
})
