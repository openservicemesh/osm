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
	// TODO(draychev): The proxy Certificate should revolve around ServiceIdentity, not specific to ServiceAccount [https://github.com/openservicemesh/osm/issues/3186]
	serviceAccount, err := catalog.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with SerialNumber=%s on Pod with UID=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil
	}

	discoveryRequest := &xds_discovery.DiscoveryRequest{
		ResourceNames: []string{
			// This cert is presented by the proxy to its peer (either upstream or downstream peer)
			// during a mTLS or TLS handshake.
			// It is referenced in the upstream cluster's UpstreamTlsContext and in the inbound
			// listener's DownstreamTlsContext.
			// The secret name is of the form <namespace>/<service-account>
			envoy.SDSCert{
				Name:     serviceAccount.String(),
				CertType: envoy.ServiceCertType,
			}.String(),

			// For each root validation cert referenced in the inbound filter chain's DownstreamTlsContext for this proxy,
			// create an SDS resource with the same name.
			// The DownstreamTlsContext on the inbound filter chain references the following validation cert types:
			// 1. root-cert-for-mtls-inbound: root validation cert to validate the downstream's cert during mTLS handshake
			// 2. root-cert-https: root validation cert to validate the downstream's cert during TLS (non-mTLS) handshake

			// This cert is the upstream service's validation certificate used to validate certificates presented
			// by downstream clients during mTLS handshake.
			// The secret name is of the form <namespace>/<upstream-service>
			envoy.SDSCert{
				Name:     serviceAccount.String(),
				CertType: envoy.RootCertTypeForMTLSInbound,
			}.String(),

			// This cert is the upstream service's validation certificate used to validate certificates presented
			// by downstream clients during TLS handshake.
			// The secret name is of the form <namespace>/<upstream-service>
			envoy.SDSCert{
				Name:     serviceAccount.String(),
				CertType: envoy.RootCertTypeForHTTPS,
			}.String(),
		},
		TypeUrl: string(envoy.TypeSDS),
	}

	// Create an SDS validation cert corresponding to each upstream service that this proxy can connect to.
	// Each cert is used to validate the certificate presented by the corresponding upstream service.
	upstreamServices := meshCatalog.ListAllowedOutboundServicesForIdentity(serviceAccount.ToServiceIdentity())
	for _, upstream := range upstreamServices {
		upstreamRootCertResource := envoy.SDSCert{
			Name:     upstream.String(),
			CertType: envoy.RootCertTypeForMTLSOutbound,
		}.String()
		discoveryRequest.ResourceNames = append(discoveryRequest.ResourceNames, upstreamRootCertResource)
	}

	return discoveryRequest
}
