package catalog

import (
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

// ExpectProxy catalogs the fact that a certificate was issued for an Envoy proxy and this is expected to connect to XDS.
func (mc *MeshCatalog) ExpectProxy(cn certificate.CommonName) {
	mc.expectedProxies.Store(cn, expectedProxy{
		certificateIssuedAt: time.Now(),
	})
}

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (mc *MeshCatalog) RegisterProxy(proxy *envoy.Proxy) {
	mc.connectedProxies.Store(proxy.GetCertificateCommonName(), connectedProxy{
		proxy:       proxy,
		connectedAt: time.Now(),
	})

	// If this proxy object is on a Kubernetes Pod - it will have an UID
	if proxy.HasPodMetadata() {
		podUID := types.UID(proxy.PodMetadata.UID)

		// Create a PodUID to Certificate CN map so we can easily determine the CN from the PodUID
		mc.podUIDToCN.Store(podUID, proxy.GetCertificateCommonName())

		// Create a PodUID to Cert Serial Number so we can easily look-up the SerialNumber of the cert issued to a proxy for a given Pod.
		mc.podUIDToCertificateSerialNumber.Store(podUID, proxy.GetCertificateSerialNumber())
	}
	log.Debug().Msgf("Registered new proxy with certificate SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (mc *MeshCatalog) UnregisterProxy(p *envoy.Proxy) {
	mc.connectedProxies.Delete(p.GetCertificateCommonName())

	mc.disconnectedProxies.Store(p.GetCertificateCommonName(), disconnectedProxy{
		lastSeen: time.Now(),
	})

	log.Debug().Msgf("Unregistered proxy with certificate SerialNumber=%v on Pod with UID=%s", p.GetCertificateSerialNumber(), p.GetPodUID())
}
