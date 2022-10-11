package envoy

import (
	"net"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/wrapperspb"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// TransportProtocolTLS is the TLS transport protocol used in Envoy configurations
	TransportProtocolTLS = "tls"

	// OutboundPassthroughCluster is the outbound passthrough cluster name
	OutboundPassthroughCluster = "passthrough-outbound"

	// StreamAccessLoggerName is name used for the envoy stream access logger
	StreamAccessLoggerName = "envoy.access_loggers.stream"
)

// ALPNInMesh indicates that the proxy is connecting to an in-mesh destination.
// It is set as a part of configuring the UpstreamTLSContext.
var ALPNInMesh = []string{"osm"}

// GetAddress creates an Envoy Address struct.
func GetAddress(address string, port uint32) *xds_core.Address {
	return &xds_core.Address{
		Address: &xds_core.Address_SocketAddress{
			SocketAddress: &xds_core.SocketAddress{
				Protocol: xds_core.SocketAddress_TCP,
				Address:  address,
				PortSpecifier: &xds_core.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
}

// GetTLSParams creates Envoy TlsParameters struct.
func GetTLSParams(sidecarSpec configv1alpha2.SidecarSpec) *xds_auth.TlsParameters {
	minVersionInt := xds_auth.TlsParameters_TlsProtocol_value[sidecarSpec.TLSMinProtocolVersion]
	maxVersionInt := xds_auth.TlsParameters_TlsProtocol_value[sidecarSpec.TLSMaxProtocolVersion]
	tlsMinVersion := xds_auth.TlsParameters_TlsProtocol(minVersionInt)
	tlsMaxVersion := xds_auth.TlsParameters_TlsProtocol(maxVersionInt)

	tlsParams := &xds_auth.TlsParameters{
		TlsMinimumProtocolVersion: tlsMinVersion,
		TlsMaximumProtocolVersion: tlsMaxVersion,
	}
	if len(sidecarSpec.CipherSuites) > 0 {
		tlsParams.CipherSuites = sidecarSpec.CipherSuites
	}
	if len(sidecarSpec.ECDHCurves) > 0 {
		tlsParams.EcdhCurves = sidecarSpec.ECDHCurves
	}

	return tlsParams
}

// getCommonTLSContext returns a CommonTlsContext type for a given 'tlsSDSCert' and 'peerValidationSDSCert' pair.
// 'tlsSDSCert' determines the SDS Secret config used to present the TLS certificate.
// 'peerValidationSDSCert' determines the SDS Secret configs used to validate the peer TLS certificate. A nil value
// is used to indicate peer certificate validation should be skipped, and is used when mTLS is disabled (ex. with TLS
// based ingress).
// 'sidecarSpec' is the sidecar section of MeshConfig.
func getCommonTLSContext(tlsSDSCert, peerValidationSDSCert string, sidecarSpec configv1alpha2.SidecarSpec) *xds_auth.CommonTlsContext {
	commonTLSContext := &xds_auth.CommonTlsContext{
		TlsParams: GetTLSParams(sidecarSpec),
		TlsCertificateSdsSecretConfigs: []*xds_auth.SdsSecretConfig{{
			// Example ==> Name: "service-cert:NameSpaceHere/ServiceNameHere"
			Name:      tlsSDSCert,
			SdsConfig: GetADSConfigSource(),
		}},
	}

	// For TLS (non-mTLS) based validation, the client certificate should not be validated and the
	// 'peerValidationSDSCert' will be set to nil to indicate this.
	if peerValidationSDSCert != "" {
		commonTLSContext.ValidationContextType = &xds_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
			ValidationContextSdsSecretConfig: &xds_auth.SdsSecretConfig{
				// Example ==> Name: "root-cert<type>:NameSpaceHere/ServiceNameHere"
				Name:      peerValidationSDSCert,
				SdsConfig: GetADSConfigSource(),
			},
		}
	}

	return commonTLSContext
}

// GetDownstreamTLSContext creates a downstream Envoy TLS Context to be configured on the upstream for the given upstream's identity
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func GetDownstreamTLSContext(upstreamIdentity identity.ServiceIdentity, mTLS bool, sidecarSpec configv1alpha2.SidecarSpec) *xds_auth.DownstreamTlsContext {
	tlsConfig := &xds_auth.DownstreamTlsContext{
		// When RequireClientCertificate is enabled trusted CA certs must be provided via ValidationContextType
		RequireClientCertificate: &wrappers.BoolValue{Value: mTLS},
	}
	if mTLS {
		// The downstream peer validation SDS cert points to a cert with the name 'upstreamIdentity' only
		// because we use a single DownstreamTlsContext for all inbound traffic to the given upstream with the identity 'upstreamIdentity'.
		// This single DownstreamTlsContext is used to validate all allowed inbound SANs. The
		// 'RootCertTypeForMTLSInbound' cert type is used for in-mesh downstreams.
		tlsConfig.CommonTlsContext = getCommonTLSContext(secrets.NameForIdentity(upstreamIdentity), secrets.NameForMTLSInbound, sidecarSpec)
	} else {
		// When 'mTLS' is disabled, the upstream should not validate the certificate presented by the downstream.
		// This is used for HTTPS ingress with mTLS disabled.
		tlsConfig.CommonTlsContext = getCommonTLSContext(secrets.NameForIdentity(upstreamIdentity), "", sidecarSpec)
	}
	return tlsConfig
}

// GetUpstreamTLSContext creates an upstream Envoy TLS Context for the given downstream identity and upstream service pair
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func GetUpstreamTLSContext(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService, sidecarSpec configv1alpha2.SidecarSpec) *xds_auth.UpstreamTlsContext {
	commonTLSContext := getCommonTLSContext(secrets.NameForIdentity(downstreamIdentity), secrets.NameForUpstreamService(upstreamSvc.Name, upstreamSvc.Namespace), sidecarSpec)

	// Advertise in-mesh using UpstreamTlsContext.CommonTlsContext.AlpnProtocols
	commonTLSContext.AlpnProtocols = ALPNInMesh
	tlsConfig := &xds_auth.UpstreamTlsContext{
		CommonTlsContext: commonTLSContext,

		// The Sni field is going to be used to do FilterChainMatch in buildInboundHTTPFilterChain()
		// The "Sni" field below of an incoming request will be matched against a list of server names
		// in FilterChainMatch.ServerNames
		Sni: upstreamSvc.ServerName(),
	}
	return tlsConfig
}

// GetADSConfigSource creates an Envoy ConfigSource struct.
func GetADSConfigSource() *xds_core.ConfigSource {
	return &xds_core.ConfigSource{
		ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
			Ads: &xds_core.AggregatedConfigSource{},
		},
		ResourceApiVersion: xds_core.ApiVersion_V3,
	}
}

// GetCIDRRangeFromStr converts the given CIDR as a string to an XDS CidrRange object
func GetCIDRRangeFromStr(cidr string) (*xds_core.CidrRange, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	prefixLen, _ := ipNet.Mask.Size()
	return &xds_core.CidrRange{
		AddressPrefix: ip.String(),
		PrefixLen: &wrapperspb.UInt32Value{
			Value: uint32(prefixLen),
		},
	}, nil
}
