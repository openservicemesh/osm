package ads

import (
	"context"
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/grpc/peer"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	serverV3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	k8s "github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/tests"
)

var kubeClient *testclient.Clientset = testclient.NewSimpleClientset()
var configClient *configFake.Clientset = configFake.NewSimpleClientset()
var meshName = "test-mesh"
var eventTimeout = 1 * time.Second
var proxySvcAccount = tests.BookstoreServiceAccount

func TestIsCNForProxy(t *testing.T) {
	assert := tassert.New(t)

	type testCase struct {
		name     string
		cn       certificate.CommonName
		proxy    *envoy.Proxy
		expected bool
	}

	certSerialNumber := certificate.SerialNumber("123456")

	testCases := []testCase{
		{
			name: "workload CN belongs to proxy",
			cn:   certificate.CommonName("svc-acc.namespace.cluster.local"),
			proxy: func() *envoy.Proxy {
				p, _ := envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.svc-acc.namespace", uuid.New(), envoy.KindSidecar)), certSerialNumber, nil)
				return p
			}(),
			expected: true,
		},
		{
			name: "workload CN does not belong to proxy",
			cn:   certificate.CommonName("svc-acc.namespace.cluster.local"),
			proxy: func() *envoy.Proxy {
				p, _ := envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.svc-acc-foo.namespace", uuid.New(), envoy.KindSidecar)), certSerialNumber, nil)
				return p
			}(),
			expected: false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			actual := isCNforProxy(tc.proxy, tc.cn)
			assert.Equal(tc.expected, actual)
		})
	}
}

// setupADSServerTestContext sets up necessary context for testing ADS Server.
// It returns a Server object, mockStream, a newly created certificate for client service and a gomock controller, which can be used for further mocks creation.
func setupADSServerTestContext(t *testing.T) (*Server, *mockStream, *x509.Certificate, *gomock.Controller) {
	assert := tassert.New(t)

	stop := signals.RegisterExitHandlers()
	kubeController, _ := k8s.NewKubernetesController(kubeClient, meshName, stop)
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockCertManager := certificate.NewMockManager(mockCtrl)

	certDuration := 1 * time.Hour

	mockConfigurator.EXPECT().IsEgressEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(certDuration).AnyTimes()
	mockConfigurator.EXPECT().IsDebugServerEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().GetMaxDataPlaneConnections().Return(0).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{}).AnyTimes()

	mc := catalog.NewFakeMeshCatalog(kubeClient, configClient)
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}))

	config := makeMockConfigWatcher()
	config.responses = makeResponses()
	s := serverV3.NewServer(context.Background(), config, serverV3.CallbackFuncs{})
	assert.NotNil(s)

	adsServer := NewADSServer(mc,
		proxyRegistry,
		true,
		tests.Namespace,
		mockConfigurator,
		mockCertManager,
		kubeController)
	assert.NotNil(adsServer)

	stream := makeMockStream(t)
	stream.recv <- &discovery.DiscoveryRequest{Node: node, TypeUrl: opaqueType}

	// test scenario when client cert not set up
	err := adsServer.StreamAggregatedResources(stream)
	assert.NotNil(err)

	// setting up client cert
	certManager := tresor.NewFakeCertManager(mockConfigurator)
	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", tests.ProxyUUID, envoy.KindSidecar, proxySvcAccount.Name, proxySvcAccount.Namespace))
	certPEM, _ := certManager.IssueCertificate(certCommonName, certDuration)
	cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())
	tests.NewFakeXDSServer(cert, nil, nil)

	return adsServer, stream, cert, mockCtrl
}

// setupClientPod creates a k8s pod running client service.
// The pod will have a envoy container running, identified by the testing proxy UUID, for the convenience of verification.
func setupClientPod(t *testing.T) *v1.Pod {
	assert := tassert.New(t)

	labels := map[string]string{constants.EnvoyUniqueIDLabelName: tests.ProxyUUID}

	namespace := tests.Namespace
	proxyUUID := tests.ProxyUUID
	proxyService := service.MeshService{Name: tests.BookstoreV1ServiceName, Namespace: namespace}
	pod := tests.NewPodFixture(namespace, fmt.Sprintf("pod-0-%s", tests.ProxyUUID), tests.BookstoreServiceAccountName, tests.PodLabels)
	pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID
	_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
	assert.Nil(err)
	svc := tests.NewServiceFixture(proxyService.Name, namespace, labels)
	_, err = kubeClient.CoreV1().Services(namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
	assert.Nil(err)
	return &pod
}

func TestStreamAggregatedResources(t *testing.T) {
	assert := tassert.New(t)

	adsServer, stream, cert, mockCtrl := setupADSServerTestContext(t)
	pod := setupClientPod(t)
	mockKubeController := k8s.NewMockController(mockCtrl)
	adsServer.kubecontroller = mockKubeController
	mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{pod})

	// ADS server starts
	stream.ctx = peer.NewContext(context.TODO(), &peer.Peer{AuthInfo: tests.NewMockAuthInfo(cert)})
	var err error
	go func() {
		err = adsServer.StreamAggregatedResources(stream)
	}()
	time.Sleep(eventTimeout) // wait for ads server to start listening

	// test event handling
	proxyBroadcastChannel := events.GetPubSubInstance().Subscribe(announcements.ProxyBroadcast)
	defer events.GetPubSubInstance().Unsub(proxyBroadcastChannel)

	certAnnouncementChannel := events.GetPubSubInstance().Subscribe(announcements.CertificateRotated)
	defer events.GetPubSubInstance().Unsub(certAnnouncementChannel)

	// test on new job creation
	tests := []struct {
		caseName               string
		triggerEvent           func()
		expectedJobsCountDiff   int
	}{
		{
			caseName: "trigger ProxyBroadcast event",
			triggerEvent: func() {
				events.GetPubSubInstance().Publish(events.PubSubMessage{
					AnnouncementType: announcements.ProxyBroadcast,
				})
			},
			expectedJobsCountDiff: 1,
		},
		{
			caseName: "trigger CertificateRotated event",
			triggerEvent: func() {
				certAnnouncementChannel <- events.PubSubMessage{
					OldObj:           nil,
					NewObj:           certificate.MockCertificater{},
					AnnouncementType: announcements.CertificateRotated,
				}
			},
			expectedJobsCountDiff: 1,
		},
	}

	for _, tc := range tests {
		prevJobsCount := adsServer.workqueues.GetJobsCount()
		t.Logf("Prev count: %v", prevJobsCount)

		tc.triggerEvent()
		select {
		case <-time.NewTimer(eventTimeout).C:
		}

		assert.Equal(prevJobsCount + tc.expectedJobsCountDiff, adsServer.workqueues.GetJobsCount())
	}

	// test ADSServer stops
	_, cancel := context.WithCancel(stream.Context())
	cancel()
	assert.Nil(err)
}
