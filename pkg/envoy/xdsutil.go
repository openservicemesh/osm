package envoy

import (
	"fmt"
	"net"
	"strings"

	xds_accesslog_filter "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_accesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// TransportProtocolTLS is the TLS transport protocol used in Envoy configurations
	TransportProtocolTLS = "tls"

	// OutboundPassthroughCluster is the outbound passthrough cluster name
	OutboundPassthroughCluster = "passthrough-outbound"

	// AccessLoggerName is name used for the envoy access loggers.
	AccessLoggerName = "envoy.access_loggers.stream"
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

// GetAccessLog creates an Envoy AccessLog struct.
func GetAccessLog() []*xds_accesslog_filter.AccessLog {
	accessLog, err := anypb.New(getStdoutAccessLog())
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling AccessLog object")
		return nil
	}
	return []*xds_accesslog_filter.AccessLog{{
		Name: AccessLoggerName,
		ConfigType: &xds_accesslog_filter.AccessLog_TypedConfig{
			TypedConfig: accessLog,
		}},
	}
}

func getStdoutAccessLog() *xds_accesslog.StdoutAccessLog {
	accessLogger := &xds_accesslog.StdoutAccessLog{
		AccessLogFormat: &xds_accesslog.StdoutAccessLog_LogFormat{
			LogFormat: &xds_core.SubstitutionFormatString{
				Format: &xds_core.SubstitutionFormatString_JsonFormat{
					JsonFormat: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"start_time":            pbStringValue(`%START_TIME%`),
							"method":                pbStringValue(`%REQ(:METHOD)%`),
							"path":                  pbStringValue(`%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%`),
							"protocol":              pbStringValue(`%PROTOCOL%`),
							"response_code":         pbStringValue(`%RESPONSE_CODE%`),
							"response_code_details": pbStringValue(`%RESPONSE_CODE_DETAILS%`),
							"time_to_first_byte":    pbStringValue(`%RESPONSE_DURATION%`),
							"upstream_cluster":      pbStringValue(`%UPSTREAM_CLUSTER%`),
							"response_flags":        pbStringValue(`%RESPONSE_FLAGS%`),
							"bytes_received":        pbStringValue(`%BYTES_RECEIVED%`),
							"bytes_sent":            pbStringValue(`%BYTES_SENT%`),
							"duration":              pbStringValue(`%DURATION%`),
							"upstream_service_time": pbStringValue(`%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%`),
							"x_forwarded_for":       pbStringValue(`%REQ(X-FORWARDED-FOR)%`),
							"user_agent":            pbStringValue(`%REQ(USER-AGENT)%`),
							"request_id":            pbStringValue(`%REQ(X-REQUEST-ID)%`),
							"requested_server_name": pbStringValue("%REQUESTED_SERVER_NAME%"),
							"authority":             pbStringValue(`%REQ(:AUTHORITY)%`),
							"upstream_host":         pbStringValue(`%UPSTREAM_HOST%`),
						},
					},
				},
			},
		},
	}
	return accessLogger
}

func pbStringValue(v string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StringValue{
			StringValue: v,
		},
	}
}

// getCommonTLSContext returns a CommonTlsContext type for a given 'tlsSDSCert' and 'peerValidationSDSCert' pair.
// 'tlsSDSCert' determines the SDS Secret config used to present the TLS certificate.
// 'peerValidationSDSCert' determines the SDS Secret configs used to validate the peer TLS certificate. A nil value
// is used to indicate peer certificate validation should be skipped, and is used when mTLS is disabled (ex. with TLS
// based ingress).
// 'sidecarSpec' is the sidecar section of MeshConfig.
func getCommonTLSContext(tlsSDSCert secrets.SDSCert, peerValidationSDSCert *secrets.SDSCert, sidecarSpec configv1alpha2.SidecarSpec) *xds_auth.CommonTlsContext {
	commonTLSContext := &xds_auth.CommonTlsContext{
		TlsParams: GetTLSParams(sidecarSpec),
		TlsCertificateSdsSecretConfigs: []*xds_auth.SdsSecretConfig{{
			// Example ==> Name: "service-cert:NameSpaceHere/ServiceNameHere"
			Name:      tlsSDSCert.String(),
			SdsConfig: GetADSConfigSource(),
		}},
	}

	// For TLS (non-mTLS) based validation, the client certificate should not be validated and the
	// 'peerValidationSDSCert' will be set to nil to indicate this.
	if peerValidationSDSCert != nil {
		commonTLSContext.ValidationContextType = &xds_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
			ValidationContextSdsSecretConfig: &xds_auth.SdsSecretConfig{
				// Example ==> Name: "root-cert<type>:NameSpaceHere/ServiceNameHere"
				Name:      peerValidationSDSCert.String(),
				SdsConfig: GetADSConfigSource(),
			},
		}
	}

	return commonTLSContext
}

// GetDownstreamTLSContext creates a downstream Envoy TLS Context to be configured on the upstream for the given upstream's identity
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func GetDownstreamTLSContext(upstreamIdentity identity.ServiceIdentity, mTLS bool, sidecarSpec configv1alpha2.SidecarSpec) *xds_auth.DownstreamTlsContext {
	upstreamSDSCert := secrets.SDSCert{
		Name:     secrets.GetSecretNameForIdentity(upstreamIdentity),
		CertType: secrets.ServiceCertType,
	}

	var downstreamPeerValidationSDSCert *secrets.SDSCert
	if mTLS {
		// The downstream peer validation SDS cert points to a cert with the name 'upstreamIdentity' only
		// because we use a single DownstreamTlsContext for all inbound traffic to the given upstream with the identity 'upstreamIdentity'.
		// This single DownstreamTlsContext is used to validate all allowed inbound SANs. The
		// 'RootCertTypeForMTLSInbound' cert type is used for in-mesh downstreams.
		downstreamPeerValidationSDSCert = &secrets.SDSCert{
			Name:     secrets.GetSecretNameForIdentity(upstreamIdentity),
			CertType: secrets.RootCertTypeForMTLSInbound,
		}
	} else {
		// When 'mTLS' is disabled, the upstream should not validate the certificate presented by the downstream.
		// This is used for HTTPS ingress with mTLS disabled.
		downstreamPeerValidationSDSCert = nil
	}

	tlsConfig := &xds_auth.DownstreamTlsContext{
		CommonTlsContext: getCommonTLSContext(upstreamSDSCert, downstreamPeerValidationSDSCert, sidecarSpec),
		// When RequireClientCertificate is enabled trusted CA certs must be provided via ValidationContextType
		RequireClientCertificate: &wrappers.BoolValue{Value: mTLS},
	}
	return tlsConfig
}

// GetUpstreamTLSContext creates an upstream Envoy TLS Context for the given downstream identity and upstream service pair
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func GetUpstreamTLSContext(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService, sidecarSpec configv1alpha2.SidecarSpec) *xds_auth.UpstreamTlsContext {
	downstreamSDSCert := secrets.SDSCert{
		Name:     secrets.GetSecretNameForIdentity(downstreamIdentity),
		CertType: secrets.ServiceCertType,
	}
	upstreamPeerValidationSDSCert := &secrets.SDSCert{
		Name:     upstreamSvc.String(),
		CertType: secrets.RootCertTypeForMTLSOutbound,
	}
	commonTLSContext := getCommonTLSContext(downstreamSDSCert, upstreamPeerValidationSDSCert, sidecarSpec)

	// Advertise in-mesh using UpstreamTlsContext.CommonTlsContext.AlpnProtocols
	commonTLSContext.AlpnProtocols = ALPNInMesh
	tlsConfig := &xds_auth.UpstreamTlsContext{
		CommonTlsContext: commonTLSContext,

		// The Sni field is going to be used to do FilterChainMatch in getInboundMeshHTTPFilterChain()
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

// GetEnvoyServiceNodeID creates the string for Envoy's "--service-node" CLI argument for the Kubernetes sidecar container Command/Args
func GetEnvoyServiceNodeID(nodeID, workloadKind, workloadName string) string {
	items := []string{
		"$(POD_UID)",
		"$(POD_NAMESPACE)",
		"$(POD_IP)",
		"$(SERVICE_ACCOUNT)",
		nodeID,
		"$(POD_NAME)",
		workloadKind,
		workloadName,
	}

	return strings.Join(items, constants.EnvoyServiceNodeSeparator)
}

// ParseEnvoyServiceNodeID parses the given Envoy service node ID and returns the encoded metadata
func ParseEnvoyServiceNodeID(serviceNodeID string) (*PodMetadata, error) {
	chunks := strings.Split(serviceNodeID, constants.EnvoyServiceNodeSeparator)

	if len(chunks) < 5 {
		return nil, fmt.Errorf("invalid envoy service node id format")
	}

	meta := &PodMetadata{
		UID:            chunks[0],
		Namespace:      chunks[1],
		IP:             chunks[2],
		ServiceAccount: identity.K8sServiceAccount{Name: chunks[3], Namespace: chunks[1]},
		EnvoyNodeID:    chunks[4],
	}

	if len(chunks) >= 8 {
		meta.Name = chunks[5]
		meta.WorkloadKind = chunks[6]
		meta.WorkloadName = chunks[7]
	}

	return meta, nil
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
