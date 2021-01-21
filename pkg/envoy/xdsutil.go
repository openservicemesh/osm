package envoy

import (
	"errors"
	"fmt"
	"strings"

	xds_accesslog_filter "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_accesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/jinzhu/copier"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
)

// SDSCertType is a type of a certificate requested by an Envoy proxy via SDS.
type SDSCertType string

// SDSDirection is a type to identify TLS certificate connectivity direction.
type SDSDirection bool

// SDSCert is only used to interface the naming and related functions to Marshal/Unmarshal a resource name,
// this avoids having sprintf/parsing logic all over the place
type SDSCert struct {
	// MeshService is a service within the mesh
	MeshService service.MeshService

	// CertType is the certificate type
	CertType SDSCertType
}

func (ct SDSCertType) String() string {
	return string(ct)
}

// SDSCertType enums
const (
	// ServiceCertType is the prefix for the service certificate resource name. Example: "service-cert:webservice"
	ServiceCertType SDSCertType = "service-cert"

	// RootCertTypeForMTLSOutbound is the prefix for the mTLS root certificate resource name for upstream connectivity. Example: "root-cert-for-mtls-outbound:webservice"
	RootCertTypeForMTLSOutbound SDSCertType = "root-cert-for-mtls-outbound"

	// RootCertTypeForMTLSInbound is the prefix for the mTLS root certificate resource name for downstream connectivity. Example: "root-cert-for-mtls-inbound:webservice"
	RootCertTypeForMTLSInbound SDSCertType = "root-cert-for-mtls-inbound"

	// RootCertTypeForHTTPS is the prefix for the HTTPS root certificate resource name. Example: "root-cert-https:webservice"
	RootCertTypeForHTTPS SDSCertType = "root-cert-https"
)

const (
	// Separator is the separator between the prefix and the name of the certificate.
	Separator = ":"

	// TransportProtocolTLS is the TLS transport protocol used in Envoy configurations
	TransportProtocolTLS = "tls"

	// OutboundPassthroughCluster is the outbound passthrough cluster name
	OutboundPassthroughCluster = "passthrough-outbound"
)

// Defines valid cert types
var validCertTypes = map[SDSCertType]interface{}{
	ServiceCertType:             nil,
	RootCertTypeForMTLSOutbound: nil,
	RootCertTypeForMTLSInbound:  nil,
	RootCertTypeForHTTPS:        nil,
}

// ALPNInMesh indicates that the proxy is connecting to an in-mesh destination.
// It is set as a part of configuring the UpstreamTLSContext.
var ALPNInMesh = []string{"osm"}

// UnmarshalSDSCert parses and returns Certificate type and a service given a
// correctly formatted string, otherwise returns error
func UnmarshalSDSCert(str string) (*SDSCert, error) {
	var svc *service.MeshService
	var ret SDSCert

	// Check separators, ignore empty string fields
	slices := strings.Split(str, Separator)
	if len(slices) != 2 {
		return nil, errInvalidCertFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.
	for _, sep := range slices {
		if len(sep) == 0 {
			return nil, errInvalidCertFormat
		}
	}

	// Check valid certType
	ret.CertType = SDSCertType(slices[0])
	if _, ok := validCertTypes[ret.CertType]; !ok {
		return nil, errInvalidCertFormat
	}

	// Check valid namespaced service name
	svc, err := service.UnmarshalMeshService(slices[1])
	if err != nil {
		return nil, err
	}
	err = copier.Copy(&ret.MeshService, &svc)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

// String is a common facility/interface to generate a string resource name out of a SDSCert
// This is to keep the sprintf logic and/or separators used agnostic to other modules
func (sdsc SDSCert) String() string {
	return fmt.Sprintf("%s%s%s",
		sdsc.CertType.String(),
		Separator,
		sdsc.MeshService.String())
}

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
func GetTLSParams() *xds_auth.TlsParameters {
	return &xds_auth.TlsParameters{
		TlsMinimumProtocolVersion: xds_auth.TlsParameters_TLSv1_2,
		TlsMaximumProtocolVersion: xds_auth.TlsParameters_TLSv1_3,
	}
}

// GetAccessLog creates an Envoy AccessLog struct.
func GetAccessLog() []*xds_accesslog_filter.AccessLog {
	accessLog, err := ptypes.MarshalAny(getFileAccessLog())
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling AccessLog object")
		return nil
	}
	return []*xds_accesslog_filter.AccessLog{{
		Name: wellknown.FileAccessLog,
		ConfigType: &xds_accesslog_filter.AccessLog_TypedConfig{
			TypedConfig: accessLog,
		}},
	}
}

func getFileAccessLog() *xds_accesslog.FileAccessLog {
	accessLogger := &xds_accesslog.FileAccessLog{
		Path: accessLogPath,
		AccessLogFormat: &xds_accesslog.FileAccessLog_LogFormat{
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
// 'peerValidationSDSCert' determines the SDS Secret configs used to validate the peer TLS certificate.
func getCommonTLSContext(tlsSDSCert, peerValidationSDSCert SDSCert) *xds_auth.CommonTlsContext {
	return &xds_auth.CommonTlsContext{
		TlsParams: GetTLSParams(),
		TlsCertificateSdsSecretConfigs: []*xds_auth.SdsSecretConfig{{
			// Example ==> Name: "service-cert:NameSpaceHere/ServiceNameHere"
			Name:      tlsSDSCert.String(),
			SdsConfig: GetADSConfigSource(),
		}},
		ValidationContextType: &xds_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
			ValidationContextSdsSecretConfig: &xds_auth.SdsSecretConfig{
				// Example ==> Name: "root-cert<type>:NameSpaceHere/ServiceNameHere"
				Name:      peerValidationSDSCert.String(),
				SdsConfig: GetADSConfigSource(),
			},
		},
	}
}

// GetDownstreamTLSContext creates a downstream Envoy TLS Context
func GetDownstreamTLSContext(upstreamSvc service.MeshService, mTLS bool) *xds_auth.DownstreamTlsContext {
	upstreamSDSCert := SDSCert{
		MeshService: upstreamSvc,
		CertType:    ServiceCertType,
	}

	var downstreamPeerValidationCertType SDSCertType
	if mTLS {
		// Perform SAN validation for downstream client certificates
		downstreamPeerValidationCertType = RootCertTypeForMTLSInbound
	} else {
		// TLS based cert validation (used for ingress)
		downstreamPeerValidationCertType = RootCertTypeForHTTPS
	}
	// The downstream peer validation SDS cert points to a cert with the name 'upstreamSvc' only
	// because we use a single DownstreamTlsContext for all inbound traffic to the given 'upstreamSvc'.
	// This single DownstreamTlsContext is used to validate all allowed inbound SANs with the
	// 'RootCertTypeForMTLSInbound' cert type used for in-mesh downstreams, while 'RootCertTypeForHTTPS'
	// cert type is used for non-mesh downstreams such as ingress.
	downstreamPeerValidationSDSCert := SDSCert{
		MeshService: upstreamSvc,
		CertType:    downstreamPeerValidationCertType,
	}

	tlsConfig := &xds_auth.DownstreamTlsContext{
		CommonTlsContext: getCommonTLSContext(upstreamSDSCert, downstreamPeerValidationSDSCert),
		// When RequireClientCertificate is enabled trusted CA certs must be provided via ValidationContextType
		RequireClientCertificate: &wrappers.BoolValue{Value: mTLS},
	}
	return tlsConfig
}

// GetUpstreamTLSContext creates an upstream Envoy TLS Context for the given downstream and upstream service pair
func GetUpstreamTLSContext(downstreamSvc, upstreamSvc service.MeshService) *xds_auth.UpstreamTlsContext {
	downstreamSDSCert := SDSCert{
		MeshService: downstreamSvc,
		CertType:    ServiceCertType,
	}
	upstreamPeerValidationSDSCert := SDSCert{
		MeshService: upstreamSvc,
		CertType:    RootCertTypeForMTLSOutbound,
	}
	commonTLSContext := getCommonTLSContext(downstreamSDSCert, upstreamPeerValidationSDSCert)

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
func GetEnvoyServiceNodeID(nodeID string) string {
	items := []string{
		"$(POD_UID)",
		"$(POD_NAMESPACE)",
		"$(POD_IP)",
		"$(SERVICE_ACCOUNT)",
		nodeID,
	}

	return strings.Join(items, constants.EnvoyServiceNodeSeparator)
}

// ParseEnvoyServiceNodeID parses the given Envoy service node ID and returns the encoded metadata
func ParseEnvoyServiceNodeID(serviceNodeID string) (*PodMetadata, error) {
	chunks := strings.Split(serviceNodeID, constants.EnvoyServiceNodeSeparator)

	if len(chunks) != 5 {
		return nil, errors.New("invalid envoy service node id format")
	}

	return &PodMetadata{
		UID:            chunks[0],
		Namespace:      chunks[1],
		IP:             chunks[2],
		ServiceAccount: chunks[3],
		EnvoyNodeID:    chunks[4],
	}, nil
}

// GetLocalClusterNameForService returns the name of the local cluster for the given service.
// The local cluster refers to the cluster corresponding to the service the proxy is fronting, accessible over localhost by the proxy.
func GetLocalClusterNameForService(proxyService service.MeshService) string {
	return GetLocalClusterNameForServiceCluster(proxyService.String())
}

// GetLocalClusterNameForServiceCluster returns the name of the local cluster for the given service cluster.
// The local cluster refers to the cluster corresponding to the service the proxy is fronting, accessible over localhost by the proxy.
func GetLocalClusterNameForServiceCluster(clusterName string) string {
	return fmt.Sprintf("%s%s", clusterName, localClusterSuffix)
}
