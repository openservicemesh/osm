package ads

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/errcode"
)

// makeRequestForAllSecrets constructs an SDS DiscoveryRequest as if an Envoy proxy sent it.
// This function is responsible for constructing an SDS DiscoveryRequest corresponding to all types of
// secrets associated with this proxy:
//
// 1. Client's service certificate when this proxy is a downstream: service-cert:<namespace>/<client-service-account-name>\
// 2. Client's root validation certificate to validate upstream services during mTLS handshake: root-cert-for-mtls-outbound:<namespace>/<server-service-name>
// 3. Server's service certificate when this proxy is an upstream: service-cert:<namespace>/<server-service-name>
// 4. Server's root validation certificate to validate downstream clients during mTLS handshake: root-cert-for-mtls-inbound:<namespace>/<server-service-name>
//
// This request will be sent to SDS which will return certificates encoded in SDS secrets corresponding to the resource names
// encoded in the DiscoveryRequest this function creates and returns.
// The proxy itself did not ask for these. We know it needs them - so we send them.
func makeRequestForAllSecrets(proxy *envoy.Proxy, meshCatalog catalog.MeshCataloger) *xds_discovery.DiscoveryRequest {
	// TODO(draychev): The proxy Certificate should revolve around ServiceIdentity, not specific to ServiceAccount [https://github.com/openservicemesh/osm/issues/3186]
	proxyIdentity, err := envoy.GetServiceIdentityFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingServiceIdentity)).
			Str("proxy", proxy.String()).Msgf("Error looking up proxy identity")
		return nil
	}

	discoveryRequest := &xds_discovery.DiscoveryRequest{
		ResourceNames: []string{
			// This cert is presented by the proxy to its peer (either upstream or downstream peer)
			// during a mTLS or TLS handshake.
			// It is referenced in the upstream cluster's UpstreamTlsContext and in the inbound
			// listener's DownstreamTlsContext.
			// The secret name is of the form <namespace>/<service-account>
			secrets.SDSCert{
				Name:     proxyIdentity.ToK8sServiceAccount().String(),
				CertType: secrets.ServiceCertType,
			}.String(),

			// For each root validation cert referenced in the inbound filter chain's DownstreamTlsContext for this proxy,
			// create an SDS resource with the same name.
			// The DownstreamTlsContext on the inbound filter chain references the following validation cert types:
			// 1. root-cert-for-mtls-inbound: root validation cert to validate the downstream's cert during mTLS handshake

			// This cert is the upstream service's validation certificate used to validate certificates presented
			// by downstream clients during mTLS handshake.
			// The secret name is of the form <namespace>/<upstream-service>
			secrets.SDSCert{
				Name:     proxyIdentity.ToK8sServiceAccount().String(),
				CertType: secrets.RootCertTypeForMTLSInbound,
			}.String(),
		},
		TypeUrl: string(envoy.TypeSDS),
	}

	// Create an SDS validation cert corresponding to each upstream service that this proxy can connect to.
	// Each cert is used to validate the certificate presented by the corresponding upstream service.
	upstreamServices := meshCatalog.ListOutboundServicesForIdentity(proxyIdentity)
	for _, upstream := range upstreamServices {
		upstreamRootCertResource := secrets.SDSCert{
			Name:     upstream.String(),
			CertType: secrets.RootCertTypeForMTLSOutbound,
		}.String()
		discoveryRequest.ResourceNames = append(discoveryRequest.ResourceNames, upstreamRootCertResource)
	}

	return discoveryRequest
}
