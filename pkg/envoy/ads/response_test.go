package ads

import (
	"context"
	"fmt"
	"time"

	mapset "github.com/deckarep/golang-set"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/metricsstore"

	"github.com/openservicemesh/osm/pkg/auth"
	catalogFake "github.com/openservicemesh/osm/pkg/catalog/fake"
	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/k8s"
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
	configClient := configFake.NewSimpleClientset()

	namespace := tests.Namespace
	proxyUUID := uuid.New()
	proxyService := service.MeshService{
		Name:      tests.BookstoreV1ServiceName,
		Namespace: namespace,
	}
	proxySvcAccount := tests.BookstoreServiceAccount

	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()

	labels := map[string]string{constants.EnvoyUniqueIDLabelName: proxyUUID.String()}
	mc := catalogFake.NewFakeMeshCatalog(kubeClient, configClient)
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}), nil)

	// Create a Pod
	pod := tests.NewPodFixture(namespace, fmt.Sprintf("pod-0-%s", uuid.New()), tests.BookstoreServiceAccountName, tests.PodLabels)
	pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()
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

	certCommonName := envoy.NewXDSCertCommonName(proxyUUID, envoy.KindSidecar, proxySvcAccount.Name, proxySvcAccount.Namespace)
	certSerialNumber := certificate.SerialNumber("123456")
	proxy, err := envoy.NewProxy(certCommonName, certSerialNumber, nil)

	Context("Proxy is valid", func() {
		Expect(proxy).ToNot((BeNil()))
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Test makeRequestForAllSecrets()", func() {
		It("returns service cert", func() {

			actual := makeRequestForAllSecrets(proxy, mc)
			expected := &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					secrets.SDSCert{
						// Proxy's own cert to present to peer during mTLS/TLS handshake
						Name:     proxySvcAccount.String(),
						CertType: secrets.ServiceCertType,
					}.String(),
					secrets.SDSCert{
						// Validation certificate for mTLS when this proxy is an upstream
						Name:     proxySvcAccount.String(),
						CertType: secrets.RootCertTypeForMTLSInbound,
					}.String(),
				},
			}
			Expect(actual).ToNot(BeNil())
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test sendAllResponses()", func() {

		certManager := tresorFake.NewFake(nil)
		certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.cluster.local", proxySvcAccount.Name, proxySvcAccount.Namespace))
		certDuration := 1 * time.Hour
		certPEM, _ := certManager.IssueCertificate(certCommonName, certDuration)
		cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())
		server, actualResponses := tests.NewFakeXDSServer(cert, nil, nil)
		kubectrlMock := k8s.NewMockController(mockCtrl)

		mockConfigurator.EXPECT().IsEgressEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
		mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(certDuration).AnyTimes()
		mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()
		mockConfigurator.EXPECT().IsDebugServerEnabled().Return(true).AnyTimes()
		mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
			EnableWASMStats:    false,
			EnableEgressPolicy: false,
		}).AnyTimes()
		mockConfigurator.EXPECT().GetMeshConfig().AnyTimes()

		metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.ProxyResponseSendSuccessCount)

		It("returns Aggregated Discovery Service response", func() {
			s := NewADSServer(mc, proxyRegistry, true, tests.Namespace, mockConfigurator, mockCertManager, kubectrlMock, nil)

			Expect(s).ToNot(BeNil())

			mockCertManager.EXPECT().IssueCertificate(gomock.Any(), certDuration).Return(certPEM, nil).Times(1)

			// Set subscribed resources for SDS
			proxy.SetSubscribedResources(envoy.TypeSDS, mapset.NewSetWith("service-cert:default/bookstore", "root-cert-for-mtls-inbound:default/bookstore"))

			err := s.sendResponse(proxy, &server, nil, mockConfigurator, envoy.XDSResponseOrder...)
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

			Expect(metricsstore.DefaultMetricsStore.Contains(fmt.Sprintf("osm_proxy_response_send_success_count{common_name=%q,type=%q} 1\n", proxy.GetCertificateCommonName(), envoy.TypeCDS))).To(BeTrue())
		})
	})

	Context("Test sendSDSResponse()", func() {

		certManager := tresorFake.NewFake(nil)
		certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", uuid.New(), envoy.KindSidecar, proxySvcAccount.Name, proxySvcAccount.Namespace))
		certDuration := 1 * time.Hour
		certPEM, _ := certManager.IssueCertificate(certCommonName, certDuration)
		cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())
		server, actualResponses := tests.NewFakeXDSServer(cert, nil, nil)
		kubectrlMock := k8s.NewMockController(mockCtrl)

		mockConfigurator.EXPECT().IsEgressEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
		mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(certDuration).AnyTimes()
		mockConfigurator.EXPECT().IsDebugServerEnabled().Return(true).AnyTimes()
		mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
			Enable: false,
		}).AnyTimes()

		It("returns Aggregated Discovery Service response", func() {
			s := NewADSServer(mc, proxyRegistry, true, tests.Namespace, mockConfigurator, mockCertManager, kubectrlMock, nil)

			Expect(s).ToNot(BeNil())

			mockCertManager.EXPECT().IssueCertificate(gomock.Any(), certDuration).Return(certPEM, nil).Times(1)

			// Set subscribed resources
			proxy.SetSubscribedResources(envoy.TypeSDS, mapset.NewSetWith("service-cert:default/bookstore", "root-cert-for-mtls-inbound:default/bookstore"))

			err := s.sendResponse(proxy, &server, nil, mockConfigurator, envoy.TypeSDS)
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
