package ads

import (
	"context"
	"fmt"
	"time"

	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test ADS response functions", func() {
	defer GinkgoRecover()

	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
		mockCertManager  *certificate.MockManager
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
	mockCertManager = certificate.NewMockManager(mockCtrl)

	// --- setup
	kubeClient := testclient.NewSimpleClientset()
	namespace := tests.Namespace
	proxyUUID := tests.ProxyUUID
	proxyService := service.MeshService{Name: tests.BookstoreV1ServiceName, Namespace: namespace}
	proxySvcAccount := tests.BookstoreServiceAccount

	labels := map[string]string{constants.EnvoyUniqueIDLabelName: tests.ProxyUUID}
	mc := catalog.NewFakeMeshCatalog(kubeClient)

	// Create a Pod
	pod := tests.NewPodFixture(namespace, fmt.Sprintf("pod-0-%s", uuid.New()), tests.BookstoreServiceAccountName, tests.PodLabels)
	pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID
	_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
	It("should have created a pod", func() {
		Expect(err).ToNot(HaveOccurred())
	})

	svc := tests.NewServiceFixture(proxyService.Name, namespace, labels)
	_, err = kubeClient.CoreV1().Services(namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
	It("should have created a service", func() {
		Expect(err).ToNot(HaveOccurred())
	})

	// Create Bookstore apex Service, since the fake catalog has a traffic split applied, needs to be
	// able to be looked up
	svc = tests.NewServiceFixture(tests.BookstoreApexService.Name, tests.BookstoreApexService.Namespace, nil)
	if _, err := kubeClient.CoreV1().Services(tests.BookstoreApexService.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("Error creating new Bookstire Apex service: %s", err.Error())
	}

	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyUUID, proxySvcAccount.Name, proxySvcAccount.Namespace))
	certSerialNumber := certificate.SerialNumber("123456")
	proxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)

	Context("Test makeRequestForAllSecrets()", func() {
		It("returns service cert", func() {

			actual := makeRequestForAllSecrets(proxy, mc)
			expected := &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					envoy.SDSCert{
						// Client certificate presented when this proxy is a downstream
						Name:     proxySvcAccount.String(),
						CertType: envoy.ServiceCertType,
					}.String(),
					envoy.SDSCert{
						// Server certificate presented when this proxy is an upstream
						Name:     proxyService.String(),
						CertType: envoy.ServiceCertType,
					}.String(),
					envoy.SDSCert{
						// Validation certificate for mTLS when this proxy is an upstream
						Name:     proxyService.String(),
						CertType: envoy.RootCertTypeForMTLSInbound,
					}.String(),
					envoy.SDSCert{
						// Validation ceritificate for TLS when this proxy is an upstream
						Name:     proxyService.String(),
						CertType: envoy.RootCertTypeForHTTPS,
					}.String(),
				},
			}
			Expect(actual).ToNot(BeNil())
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test sendAllResponses()", func() {

		certManager := tresor.NewFakeCertManager(mockConfigurator)
		certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New(), proxySvcAccount.Name, proxySvcAccount.Namespace))
		certDuration := 1 * time.Hour
		certPEM, _ := certManager.IssueCertificate(certCommonName, certDuration)
		cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())
		server, actualResponses := tests.NewFakeXDSServer(cert, nil, nil)

		mockConfigurator.EXPECT().IsEgressEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsPrometheusScrapingEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
		mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(certDuration).AnyTimes()
		mockConfigurator.EXPECT().IsDebugServerEnabled().Return(true).AnyTimes()

		It("returns Aggregated Discovery Service response", func() {
			s := NewADSServer(mc, true, tests.Namespace, mockConfigurator, mockCertManager)

			Expect(s).ToNot(BeNil())

			mockCertManager.EXPECT().IssueCertificate(gomock.Any(), certDuration).Return(certPEM, nil).Times(1)
			s.sendAllResponses(proxy, &server, mockConfigurator)

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

			// Expect 4 SDS certs:
			// 1. client cert when this proxy is a downstream
			// 2. mTLS validation cert when this proxy is an upstream
			// 3. TLS validation cert when this proxy is an upstream
			// 4. server cert when this proxy is an upstream
			Expect(len((*actualResponses)[4].Resources)).To(Equal(4))

			var tmpResource *any.Any

			clientServiceCert := xds_auth.Secret{}
			tmpResource = (*actualResponses)[4].Resources[0]
			err = ptypes.UnmarshalAny(tmpResource, &clientServiceCert)
			Expect(err).To(BeNil())
			Expect(clientServiceCert.Name).To(Equal(envoy.SDSCert{
				Name:     proxySvcAccount.String(),
				CertType: envoy.ServiceCertType,
			}.String()))

			serverServiceCert := xds_auth.Secret{}
			tmpResource = (*actualResponses)[4].Resources[1]
			err = ptypes.UnmarshalAny(tmpResource, &serverServiceCert)
			Expect(err).To(BeNil())
			Expect(serverServiceCert.Name).To(Equal(envoy.SDSCert{
				Name:     proxyService.String(),
				CertType: envoy.ServiceCertType,
			}.String()))

			serverRootCertTypeForMTLSInbound := xds_auth.Secret{}
			tmpResource = (*actualResponses)[4].Resources[2]
			err = ptypes.UnmarshalAny(tmpResource, &serverRootCertTypeForMTLSInbound)
			Expect(err).To(BeNil())
			Expect(serverRootCertTypeForMTLSInbound.Name).To(Equal(envoy.SDSCert{
				Name:     proxyService.String(),
				CertType: envoy.RootCertTypeForMTLSInbound,
			}.String()))

			serverRootCertTypeForHTTPS := xds_auth.Secret{}
			tmpResource = (*actualResponses)[4].Resources[3]
			err = ptypes.UnmarshalAny(tmpResource, &serverRootCertTypeForHTTPS)
			Expect(err).To(BeNil())
			Expect(serverRootCertTypeForHTTPS.Name).To(Equal(envoy.SDSCert{
				Name:     proxyService.String(),
				CertType: envoy.RootCertTypeForHTTPS,
			}.String()))
		})
	})

	Context("Test sendSDSResponse()", func() {

		certManager := tresor.NewFakeCertManager(mockConfigurator)
		certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New(), proxySvcAccount.Name, proxySvcAccount.Namespace))
		certDuration := 1 * time.Hour
		certPEM, _ := certManager.IssueCertificate(certCommonName, certDuration)
		cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())
		server, actualResponses := tests.NewFakeXDSServer(cert, nil, nil)

		mockConfigurator.EXPECT().IsEgressEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsPrometheusScrapingEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
		mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(certDuration).AnyTimes()
		mockConfigurator.EXPECT().IsDebugServerEnabled().Return(true).AnyTimes()
		mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(configurator.ExternAuthConfig{
			Enable: false,
		}).AnyTimes()

		It("returns Aggregated Discovery Service response", func() {
			s := NewADSServer(mc, true, tests.Namespace, mockConfigurator, mockCertManager)

			Expect(s).ToNot(BeNil())

			mockCertManager.EXPECT().IssueCertificate(gomock.Any(), certDuration).Return(certPEM, nil).Times(1)
			s.sendSDSResponse(proxy, &server, mockConfigurator)

			Expect(actualResponses).ToNot(BeNil())
			Expect(len(*actualResponses)).To(Equal(1))

			sdsResponse := (*actualResponses)[0]

			Expect(sdsResponse.VersionInfo).To(Equal("2")) // 2 because first update was by the previous test for the proxy
			Expect(sdsResponse.TypeUrl).To(Equal(string(envoy.TypeSDS)))

			// Expect 4 SDS certs:
			// 1. client cert when this proxy is a downstream
			// 2. mTLS validation cert when this proxy is an upstream
			// 3. TLS validation cert when this proxy is an upstream
			// 4. server cert when this proxy is an upstream
			Expect(len(sdsResponse.Resources)).To(Equal(4))

			var tmpResource *any.Any

			clientServiceCert := xds_auth.Secret{}
			tmpResource = sdsResponse.Resources[0]
			err = ptypes.UnmarshalAny(tmpResource, &clientServiceCert)
			Expect(err).To(BeNil())
			Expect(clientServiceCert.Name).To(Equal(envoy.SDSCert{
				Name:     proxySvcAccount.String(),
				CertType: envoy.ServiceCertType,
			}.String()))

			serverServiceCert := xds_auth.Secret{}
			tmpResource = sdsResponse.Resources[1]
			err = ptypes.UnmarshalAny(tmpResource, &serverServiceCert)
			Expect(err).To(BeNil())
			Expect(serverServiceCert.Name).To(Equal(envoy.SDSCert{
				Name:     proxyService.String(),
				CertType: envoy.ServiceCertType,
			}.String()))

			serverRootCertTypeForMTLSInbound := xds_auth.Secret{}
			tmpResource = sdsResponse.Resources[2]
			err = ptypes.UnmarshalAny(tmpResource, &serverRootCertTypeForMTLSInbound)
			Expect(err).To(BeNil())
			Expect(serverRootCertTypeForMTLSInbound.Name).To(Equal(envoy.SDSCert{
				Name:     proxyService.String(),
				CertType: envoy.RootCertTypeForMTLSInbound,
			}.String()))

			serverRootCertTypeForHTTPS := xds_auth.Secret{}
			tmpResource = sdsResponse.Resources[3]
			err = ptypes.UnmarshalAny(tmpResource, &serverRootCertTypeForHTTPS)
			Expect(err).To(BeNil())
			Expect(serverRootCertTypeForHTTPS.Name).To(Equal(envoy.SDSCert{
				Name:     proxyService.String(),
				CertType: envoy.RootCertTypeForHTTPS,
			}.String()))
		})
	})
})
