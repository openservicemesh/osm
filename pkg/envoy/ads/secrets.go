package ads

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/envoy"
)

// makeRequestForAllSecrets constructs an SDS DiscoveryRequest as if an Envoy proxy sent it.
// This function is responsible for constructing an SDS DiscoveryRequest corresponding to all types of
// secrets associated with this proxy:
//
// 1. Client's service certificate when this proxy is a downstream: service-cert:<namespace>/<client-service-account-name>\
// 2. Client's root validation certificate to validate upstream services during mTLS handshake: root-cert-for-mtls-outbound:<namespace>/<server-service-name>
// 3. Server's service certificate when this proxy is an upstream: service-cert:<namespace>/<server-service-name>
// 4. Server's root validation certificate to validate downstream clients during mTLS handshake: root-cert-for-mtls-inbound:<namespace>/<server-service-name>
// 5. Server's root validation certificate to validate downstream clients during TLS handshake: root-cert-https:<namespace>/<server-service-name>
//
// This request will be sent to SDS which will return certificates encoded in SDS secrets corresponding to the resource names
// encoded in the DiscoveryRequest this function creates and returns.
// The proxy itself did not ask for these. We know it needs them - so we send them.
func makeRequestForAllSecrets(proxy *envoy.Proxy, meshCatalog catalog.MeshCataloger) *xds_discovery.DiscoveryRequest {
	proxyIdentity, err := catalog.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with SerialNumber=%s on Pod with UID=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil
	}

	discoveryRequest := &xds_discovery.DiscoveryRequest{
		ResourceNames: []string{
			// This cert is the client certificate presented by the proxy when it is connecting to
			// an upstream service. This secret is referenced in the UpstreamTlsContext for any
			// upstream TLS cluster configured for this proxy.
			// The secret name is of the form <namespace>/<client-service-account>
			envoy.SDSCert{
				Name:     proxyIdentity.String(),
				CertType: envoy.ServiceCertType,
			}.String(),
		},
		TypeUrl: string(envoy.TypeSDS),
	}

	// Create an SDS validation cert corresponding to each upstream service that this proxy can connect to.
	// Each cert is used to validate the certificate presented by the corresponding upstream service.
	upstreamServices := meshCatalog.ListAllowedOutboundServicesForIdentity(proxyIdentity)
	for _, upstream := range upstreamServices {
		upstreamRootCertResource := envoy.SDSCert{
			Name:     upstream.String(),
			CertType: envoy.RootCertTypeForMTLSOutbound,
		}.String()
		discoveryRequest.ResourceNames = append(discoveryRequest.ResourceNames, upstreamRootCertResource)
	}

	// Create an SDS service cert and validation cert for each service associated with this proxy.
	// The service cert and validation cert are referenced in the per service inbound filter chain's DownstreamTlsContext
	// that is a part the proxy's inbound listener.
	proxyServices, err := meshCatalog.GetServicesFromEnvoyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error getting services associated with Envoy with certificate SerialNumber=%s on Pod with UID=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil
	}
	for _, proxySvc := range proxyServices {
		upstreamSecretResourceNames := []string{
			// This cert is the upsteram service's service certificate that is presented to downstream
			// clients during mTLS or TLS handshake.
			// It is of the form <namespace>/<upstream-service>
			envoy.SDSCert{
				Name:     proxySvc.String(),
				CertType: envoy.ServiceCertType,
			}.String(),

			// This cert is the upstream service's validation certificate used to validate certificates presented
			// by downstream clients during mTLS handshake.
			// The secret name is of the form <namespace>/<upstream-service>
			envoy.SDSCert{
				Name:     proxySvc.String(),
				CertType: envoy.RootCertTypeForMTLSInbound,
			}.String(),

			// This cert is the upstream service's validation certificate used to validate certificates presented
			// by downstream clients during TLS handshake.
			// The secret name is of the form <namespace>/<upstream-service>
			envoy.SDSCert{
				Name:     proxySvc.String(),
				CertType: envoy.RootCertTypeForHTTPS,
			}.String(),
		}

		discoveryRequest.ResourceNames = append(discoveryRequest.ResourceNames, upstreamSecretResourceNames...)
	}

	return discoveryRequest
}
