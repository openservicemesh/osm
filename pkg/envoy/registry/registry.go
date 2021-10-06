package registry

import (
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// NewProxyRegistry initializes a new empty *ProxyRegistry.
func NewProxyRegistry(mapper ProxyServiceMapper, msgBroker *messaging.Broker) *ProxyRegistry {
	return &ProxyRegistry{
		ProxyServiceMapper: mapper,
		msgBroker:          msgBroker,
	}
}

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (pr *ProxyRegistry) RegisterProxy(proxy *envoy.Proxy) {
	pr.connectedProxies.Store(proxy.GetCertificateCommonName(), connectedProxy{
		proxy:       proxy,
		connectedAt: time.Now(),
	})

	// If this proxy object is on a Kubernetes Pod - it will have an UID
	if proxy.HasPodMetadata() {
		podUID := types.UID(proxy.PodMetadata.UID)

		// Create a PodUID to Certificate CN map so we can easily determine the CN from the PodUID
		pr.podUIDToCN.Store(podUID, proxy.GetCertificateCommonName())

		// Create a PodUID to Cert Serial Number so we can easily look-up the SerialNumber of the cert issued to a proxy for a given Pod.
		pr.podUIDToCertificateSerialNumber.Store(podUID, proxy.GetCertificateSerialNumber())
	}
	log.Debug().Str("proxy", proxy.String()).Msg("Registered new proxy")
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (pr *ProxyRegistry) UnregisterProxy(p *envoy.Proxy) {
	pr.connectedProxies.Delete(p.GetCertificateCommonName())
	log.Debug().Msgf("Unregistered proxy %s", p.String())
}

// GetConnectedProxyCount counts the number of connected proxies
func (pr *ProxyRegistry) GetConnectedProxyCount() int {
	return len(pr.ListConnectedProxies())
}
