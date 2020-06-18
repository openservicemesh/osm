package ads

import (
	"context"
	"fmt"
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/google/uuid"
	"google.golang.org/grpc/peer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("Test StreamAggregatedResources XDS implementation", func() {

	getProxy := func(connectedProxies map[certificate.CommonName]*envoy.Proxy) *envoy.Proxy {
		var connectedProxiesList []*envoy.Proxy
		for _, proxy := range connectedProxies {
			connectedProxiesList = append(connectedProxiesList, proxy)
		}
		if connectedProxiesList != nil {
			return connectedProxiesList[0]
		}
		return nil
	}

	kubeClient := testclient.NewSimpleClientset()
	mc := catalog.NewFakeMeshCatalog(kubeClient)
	proxyID := uuid.New().String()

	// Create the Kubernetes POD
	{
		pod := tests.NewPodTestFixture(tests.Namespace, uuid.New().String())
		pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyID
		pod.Labels[tests.SelectorKey] = tests.SelectorValue
		_, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
		It("Created the pod", func() {
			Expect(err).ToNot(HaveOccurred())
		})

	}

	// Create the Kubernetes SERVICE
	{

		svc := tests.NewServiceFixture(tests.BookstoreServiceName, tests.Namespace, map[string]string{tests.SelectorKey: tests.SelectorValue})
		_, err := kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		It("Created the pod", func() {
			Expect(err).ToNot(HaveOccurred())
		})
	}

	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyID, tests.BookbuyerServiceAccountName, tests.Namespace))
	certPEM, _ := certManager.IssueCertificate(cn, nil)

	cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())

	// Send DiscoveryRequests to this channel - this will make it as if an Envoy proxy sent a request to OSM
	fromEnvoyToOSM := make(chan envoy_api_v2.DiscoveryRequest)

	// OSM responses to the Envoy proxy would end up here
	responseCh := make(chan envoy_api_v2.DiscoveryResponse)

	xds, actualResponses := tests.NewFakeXDSServer(cert, fromEnvoyToOSM, responseCh)

	adsServer := Server{
		ctx: peer.NewContext(context.TODO(), &peer.Peer{
			Addr:     tests.NewMockAddress("9.8.7.6"),
			AuthInfo: tests.NewMockAuthInfo(cert),
		}),
		catalog:     mc,
		meshSpec:    smi.NewFakeMeshSpecClient(),
		xdsHandlers: getHandlers(),
	}

	// Start the server!!
	go func() { _ = adsServer.StreamAggregatedResources(xds) }()

	Context("Test Envoy XDS GRPC .Recv() and .Send()", func() {

		It("handles NACK (DiscoveryRequest with an Error)", func() {
			// Make a NACK and send it to OSM
			nack := tests.NewDiscoveryRequestWithError(envoy.TypeSDS.String(), "aa", "failed to apply SDS config", uint64(123))
			fromEnvoyToOSM <- *nack

			// There were NO responses back from OSM because
			Expect(len(*actualResponses)).To(Equal(0))
			Expect(len(mc.ListConnectedProxies())).To(Equal(1))

			proxy := getProxy(mc.ListConnectedProxies())

			lastAppliedSDS := proxy.GetLastAppliedVersion(envoy.TypeSDS)
			Expect(lastAppliedSDS).To(Equal(uint64(0)))

			Expect(len(mc.ListExpectedProxies())).To(Equal(0))
			Expect(len(mc.ListDisconnectedProxies())).To(Equal(0))

			// Send a request with a new higher VersionInfo
			req := tests.NewDiscoveryRequest(envoy.TypeSDS.String(), "yy", uint64(234))
			fromEnvoyToOSM <- *req

			response := <-responseCh

			// Should have increased VersionInfo in the OSM->Envoy response +1
			Expect(response.TypeUrl).To(Equal(envoy.TypeSDS.String()))
			Expect(response.VersionInfo).To(Equal("235"))
			Expect(len(mc.ListConnectedProxies())).To(Equal(1))

			Expect(proxy.GetLastAppliedVersion(envoy.TypeSDS)).To(Equal(uint64(234)))
			Expect(proxy.GetLastSentVersion(envoy.TypeSDS)).To(Equal(uint64(235)))
			Expect(len(*actualResponses)).To(Equal(1))

			// ACK version 235

			req = tests.NewDiscoveryRequest(envoy.TypeSDS.String(), "xx", uint64(235))
			fromEnvoyToOSM <- *req

			Expect(proxy.GetLastAppliedVersion(envoy.TypeSDS)).To(Equal(uint64(234)))
			Expect(len(*actualResponses)).To(Equal(1))
		})

	})
})
