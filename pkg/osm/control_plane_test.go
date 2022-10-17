package osm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/models"
)

type fakeConfig string

type fakeGenerator struct {
	mu        sync.Mutex
	callCount map[string]int
}

func (g *fakeGenerator) getCallCount(uuid string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.callCount[uuid]
}

func (g *fakeGenerator) GenerateConfig(ctx context.Context, proxy *models.Proxy) (fakeConfig, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.callCount[proxy.UUID.String()]++
	return fakeConfig(fmt.Sprintf("%s: %d", proxy.UUID, g.callCount[proxy.UUID.String()])), nil
}

type fakeServer struct {
	mu             sync.Mutex
	proxyConfigMap map[string]fakeConfig

	callCount map[string]int
}

func (s *fakeServer) getConfig(uuid string) fakeConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.proxyConfigMap[uuid]
}

func (s *fakeServer) getCallCount(uuid string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callCount[uuid]
}

func (s *fakeServer) UpdateProxy(ctx context.Context, proxy *models.Proxy, config fakeConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callCount[proxy.UUID.String()]++
	s.proxyConfigMap[proxy.UUID.String()] = config

	return nil
}

func TestControlLoop(t *testing.T) {
	tassert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	server := &fakeServer{
		proxyConfigMap: make(map[string]fakeConfig),
		callCount:      make(map[string]int),
	}
	g := &fakeGenerator{
		callCount: make(map[string]int),
	}
	certManager := tresorFake.NewFake(1 * time.Hour)
	stop := make(chan struct{})

	provider := compute.NewMockInterface(mockCtrl)

	provider.EXPECT().GetMeshConfig().AnyTimes()
	provider.EXPECT().VerifyProxy(gomock.Any()).AnyTimes()

	meshCatalog := catalog.NewMeshCatalog(
		provider,
		tresorFake.NewFake(time.Hour),
		stop,
		messaging.NewBroker(stop),
	)

	cp := NewControlPlane[fakeConfig](server, g, meshCatalog, registry.NewProxyRegistry(), certManager, messaging.NewBroker(stop))

	// With no proxies registered, should be empty
	time.Sleep(time.Second)
	tassert.Empty(server.proxyConfigMap)
	tassert.Empty(server.callCount)
	tassert.Empty(g.callCount)

	uuid1 := uuid.New()
	id1 := identity.New("p2", "ns2")
	cert1, err := certManager.IssueCertificate(certificate.ForCommonNamePrefix(models.NewXDSCertCNPrefix(uuid1, models.KindSidecar, id1)))
	tassert.NoError(err)

	uuid2 := uuid.New()
	id2 := identity.New("p2", "ns2")
	cert2, err := certManager.IssueCertificate(certificate.ForCommonNamePrefix(models.NewXDSCertCNPrefix(uuid2, models.KindSidecar, id2)))
	tassert.NoError(err)

	x509Cert1, err := certificate.DecodePEMCertificate(cert1.GetCertificateChain())
	tassert.NoError(err)

	x509Cert2, err := certificate.DecodePEMCertificate(cert2.GetCertificateChain())
	tassert.NoError(err)

	// Register p1
	ctx1 := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{
				// TODO: write in the fake cert.
				VerifiedChains: [][]*x509.Certificate{{x509Cert1}},
			},
		},
	})

	ctx2 := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{
				// TODO: write in the fake cert.
				VerifiedChains: [][]*x509.Certificate{{x509Cert2}},
			},
		},
	})

	// Connect proxy 1
	err = cp.ProxyConnected(ctx1, 1)
	tassert.NoError(err)

	time.Sleep(time.Second)

	// p1 got registered
	p1 := cp.proxyRegistry.GetConnectedProxy(1)
	tassert.NotNil(p1)
	tassert.Equal(uuid1, p1.UUID)
	tassert.Equal(id1, p1.Identity)
	tassert.Equal(1, cp.proxyRegistry.GetConnectedProxyCount())

	// p1 got a call to config generation
	tassert.Equal(1, g.getCallCount(p1.UUID.String()))
	tassert.Equal(1, server.getCallCount(p1.UUID.String()))
	tassert.Equal(fakeConfig(p1.UUID.String()+": 1"), server.getConfig(p1.UUID.String()))

	// Broadcast an update for proxy 1
	cp.msgBroker.BroadcastProxyUpdate()
	// need to wait at least 2 seconds for the sliding window in the message broker.
	time.Sleep(time.Second * 3)

	// server and config generation got called again
	tassert.Equal(2, g.getCallCount(p1.UUID.String()))
	tassert.Equal(2, server.getCallCount(p1.UUID.String()))
	tassert.Equal(fakeConfig(p1.UUID.String()+": 2"), server.getConfig(p1.UUID.String()))

	err = cp.ProxyConnected(ctx2, 2)
	tassert.NoError(err)

	time.Sleep(time.Second)

	// p2 got registered
	p2 := cp.proxyRegistry.GetConnectedProxy(2)
	tassert.NotNil(p2)
	tassert.Equal(uuid2, p2.UUID)
	tassert.Equal(id2, p2.Identity)
	tassert.Equal(2, cp.proxyRegistry.GetConnectedProxyCount())

	// p1 did not get an update from another proxy connecting.
	tassert.Equal(2, g.getCallCount(p1.UUID.String()))
	tassert.Equal(2, server.getCallCount(p1.UUID.String()))
	tassert.Equal(fakeConfig(p1.UUID.String()+": 2"), server.getConfig(p1.UUID.String()))

	//p2 got a call for config generation
	tassert.Equal(1, g.getCallCount(p2.UUID.String()))
	tassert.Equal(1, server.getCallCount(p2.UUID.String()))
	tassert.Equal(fakeConfig(p2.UUID.String()+": 1"), server.getConfig(p2.UUID.String()))

	// broadcast a second update and verify that both have gotten it.
	cp.msgBroker.BroadcastProxyUpdate()
	time.Sleep(time.Second * 3)

	//p2 got a call for config generation
	tassert.Equal(2, g.getCallCount(p2.UUID.String()))
	tassert.Equal(2, server.getCallCount(p2.UUID.String()))
	tassert.Equal(fakeConfig(p2.UUID.String()+": 2"), server.getConfig(p2.UUID.String()))

	// p1 got a call to config generation
	tassert.Equal(3, g.getCallCount(p1.UUID.String()))
	tassert.Equal(3, server.getCallCount(p1.UUID.String()))
	tassert.Equal(fakeConfig(p1.UUID.String()+": 3"), server.getConfig(p1.UUID.String()))
}
