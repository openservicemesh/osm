package ads

import (
	"context"
	"fmt"
	"time"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"google.golang.org/grpc/peer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test StreamAggregatedResources XDS implementation", func() {
	defer GinkgoRecover()

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

	{ // Create the Kubernetes POD
		labels := map[string]string{
			constants.EnvoyUniqueIDLabelName: proxyID,
			tests.SelectorKey:                tests.SelectorValue,
		}
		pod := tests.NewPodFixture(tests.Namespace, uuid.New().String(), tests.BookstoreServiceAccountName, labels)
		newPod, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
		It("Created the pod", func() { Expect(err).ToNot(HaveOccurred()) })
		log.Info().Msgf(">>> Created Test Pod: %s/%s", newPod.Namespace, newPod.Name)
	}

	{ // Create the Kubernetes SERVICE
		svc := tests.NewServiceFixture(tests.BookstoreV1ServiceName, tests.Namespace, map[string]string{tests.SelectorKey: tests.SelectorValue})
		newService, err := kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		It("Created the service", func() { Expect(err).ToNot(HaveOccurred()) })
		log.Info().Msgf(">>> Created Test Service: %s/%s", newService.Namespace, newService.Name)
	}

	mockCtrl := gomock.NewController(GinkgoT())
	defer mockCtrl.Finish()
	cfg := configurator.NewMockConfigurator(mockCtrl) // .NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
	cfg.EXPECT().IsDebugServerEnabled().AnyTimes().Return(false)
	cfg.EXPECT().IsPermissiveTrafficPolicyMode().AnyTimes().Return(false)
	cfg.EXPECT().IsEgressEnabled().AnyTimes().Return(false)
	cfg.EXPECT().IsPrometheusScrapingEnabled().AnyTimes().Return(false)
	cfg.EXPECT().IsTracingEnabled().AnyTimes().Return(false)

	// cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(cfg) // &cache,
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyID, tests.BookbuyerServiceAccountName, tests.Namespace))
	certPEM, _ := certManager.IssueCertificate(cn, 9*time.Minute)

	cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())

	// Send DiscoveryRequests to this channel - this will make it as if an Envoy proxy sent a request to OSM
	fromEnvoyToOSM := make(chan *xds_discovery.DiscoveryRequest)

	// OSM responses to the Envoy proxy would end up here
	responseCh := make(chan *xds_discovery.DiscoveryResponse)

	// This returns a new XDSServer, which implements AggregatedDiscoveryService_StreamAggregatedResourcesServer.
	// It will implement Send() and Recv(), which leverage the responsesCh and requestsCh channels.
	xds, actualResponses := tests.NewFakeXDSServer(cert, fromEnvoyToOSM, responseCh)

	adsServer := Server{
		ctx: peer.NewContext(context.TODO(), &peer.Peer{
			Addr:     tests.NewMockAddress("9.8.7.6"),
			AuthInfo: tests.NewMockAuthInfo(cert),
		}),
		catalog:     mc,
		meshSpec:    smi.NewFakeMeshSpecClient(),
		xdsHandlers: getHandlers(),
		cfg:         cfg,
	}

	// Start the server!!
	go func() {
		log.Info().Msg(">>> Starting StreamAggregatedResources goroutine...")
		defer GinkgoRecover()
		_ = adsServer.StreamAggregatedResources(xds)
	}()

	Context("Test Envoy XDS GRPC .Recv() and .Send()", func() {

		It("handles NACK (DiscoveryRequest with an Error)", func() {
			// Make a NACK and send it to OSM
			versionInfo := 123
			log.Info().Msgf(">>> Sending a NACK with VersionInfo: %d", versionInfo)
			nack := tests.NewDiscoveryRequestWithError(envoy.TypeSDS, "aa", "failed to apply SDS config", uint64(versionInfo))
			fromEnvoyToOSM <- nack
			log.Info().Msg(">>> Sent a NACK!")

			// There were NO responses back from OSM because
			/*
				Expect(len(*actualResponses)).To(Equal(0),
					fmt.Sprintf("Expected 0 responses so far; found %d: %+v", len(*actualResponses), *actualResponses))
			*/
			Expect(len(mc.ListConnectedProxies())).To(Equal(1),
				fmt.Sprintf("Expected connected proxies to be 1; Got %d instead: %+v", len(mc.ListConnectedProxies()), mc.ListConnectedProxies()))

			log.Info().Msg(">>> Get the connected proxy...")
			proxy := getProxy(mc.ListConnectedProxies())

			log.Info().Msg(">>> Get Last Applied Version...")
			lastAppliedSDS := proxy.GetLastAppliedVersion(envoy.TypeSDS)
			Expect(lastAppliedSDS).To(Equal(uint64(0)))

			Expect(len(mc.ListExpectedProxies())).To(Equal(0))
			Expect(len(mc.ListDisconnectedProxies())).To(Equal(0))

			higherVersionInfo := 234
			log.Info().Msgf(">>> Sending a request with a higher VersionInfo: %d", higherVersionInfo)
			// Send a request with a new higher VersionInfo
			req := tests.NewDiscoveryRequest(envoy.TypeSDS, "yy", uint64(higherVersionInfo))
			fromEnvoyToOSM <- req

			log.Info().Msg(">>> Responding to the request with a higher VersionInfo...")
			response := <-responseCh

			// Should have increased VersionInfo in the OSM->Envoy response +1
			Expect(response.TypeUrl).To(Equal(envoy.TypeSDS.String()))
			Expect(response.VersionInfo).To(Equal("235"))
			Expect(len(mc.ListConnectedProxies())).To(Equal(1))

			Expect(proxy.GetLastAppliedVersion(envoy.TypeSDS)).To(Equal(uint64(234)))
			Expect(proxy.GetLastSentVersion(envoy.TypeSDS)).To(Equal(uint64(235)))
			Expect(len(*actualResponses)).To(Equal(1))

			// ACK version 235
			log.Info().Msgf("Ack version 235")
			req = tests.NewDiscoveryRequest(envoy.TypeSDS, "xx", uint64(235))
			fromEnvoyToOSM <- req

			Expect(proxy.GetLastAppliedVersion(envoy.TypeSDS)).To(Equal(uint64(234)))
			Expect(len(*actualResponses)).To(Equal(1))
		})
	})
})
