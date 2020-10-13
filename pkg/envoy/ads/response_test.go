package ads

import (
	"context"
	"fmt"
	"time"

	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
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
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	// --- setup
	kubeClient := testclient.NewSimpleClientset()
	namespace := tests.Namespace
	envoyUID := tests.EnvoyUID
	serviceName := tests.BookstoreV1ServiceName
	serviceAccountName := tests.BookstoreServiceAccountName

	labels := map[string]string{constants.EnvoyUniqueIDLabelName: tests.EnvoyUID}
	mc := catalog.NewFakeMeshCatalog(kubeClient)

	// Create a Pod
	pod := tests.NewPodTestFixture(namespace, fmt.Sprintf("pod-0-%s", uuid.New()))
	pod.Labels[constants.EnvoyUniqueIDLabelName] = envoyUID
	_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
	It("should have created a pod", func() {
		Expect(err).ToNot(HaveOccurred())
	})

	svc := tests.NewServiceFixture(serviceName, namespace, labels)
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

	cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", envoyUID, serviceAccountName, namespace))
	proxy := envoy.NewProxy(cn, nil)

	meshService := service.MeshService{
		Namespace: "default",
		Name:      serviceName,
	}

	Context("Test getRequestedCertType()", func() {
		It("returns service cert", func() {

			actual := makeRequestForAllSecrets(proxy, mc)
			expected := &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					envoy.SDSCert{
						MeshService: meshService,
						CertType:    envoy.ServiceCertType,
					}.String(),
					envoy.SDSCert{
						MeshService: meshService,
						CertType:    envoy.RootCertTypeForMTLSOutbound,
					}.String(),
					envoy.SDSCert{
						MeshService: meshService,
						CertType:    envoy.RootCertTypeForMTLSInbound,
					}.String(),
					envoy.SDSCert{
						MeshService: meshService,
						CertType:    envoy.RootCertTypeForHTTPS,
					}.String(),
				},
			}
			Expect(actual).ToNot(BeNil())
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test sendAllResponses()", func() {

		cache := make(map[certificate.CommonName]certificate.Certificater)
		certManager := tresor.NewFakeCertManager(&cache, mockConfigurator)
		cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New(), serviceAccountName, tests.Namespace))
		certPEM, _ := certManager.IssueCertificate(cn, 1*time.Hour)
		cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())
		server, actualResponses := tests.NewFakeXDSServer(cert, nil, nil)

		mockConfigurator.EXPECT().IsEgressEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsPrometheusScrapingEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
		mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

		It("returns Aggregated Discovery Service response", func() {
			s := NewADSServer(mc, true, tests.Namespace, mockConfigurator)

			Expect(s).ToNot(BeNil())

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
			Expect(len((*actualResponses)[4].Resources)).To(Equal(4))

			secretOne := xds_auth.Secret{}
			firstSecret := (*actualResponses)[4].Resources[0]
			err = ptypes.UnmarshalAny(firstSecret, &secretOne)
			Expect(secretOne.Name).To(Equal(envoy.SDSCert{
				MeshService: meshService,
				CertType:    envoy.ServiceCertType,
			}.String()))

			secretTwo := xds_auth.Secret{}
			secondSecret := (*actualResponses)[4].Resources[1]
			err = ptypes.UnmarshalAny(secondSecret, &secretTwo)
			Expect(secretTwo.Name).To(Equal(envoy.SDSCert{
				MeshService: meshService,
				CertType:    envoy.RootCertTypeForMTLSOutbound,
			}.String()))

			secretThree := xds_auth.Secret{}
			thirdSecret := (*actualResponses)[4].Resources[2]
			err = ptypes.UnmarshalAny(thirdSecret, &secretThree)
			Expect(secretThree.Name).To(Equal(envoy.SDSCert{
				MeshService: meshService,
				CertType:    envoy.RootCertTypeForMTLSInbound,
			}.String()))

			secretFour := xds_auth.Secret{}
			forthSecret := (*actualResponses)[4].Resources[3]
			err = ptypes.UnmarshalAny(forthSecret, &secretFour)
			Expect(secretFour.Name).To(Equal(envoy.SDSCert{
				MeshService: meshService,
				CertType:    envoy.RootCertTypeForHTTPS,
			}.String()))
		})
	})
})
