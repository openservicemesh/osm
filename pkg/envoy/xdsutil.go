package envoy

import (
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	envoy_config_filter_accesslog_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/open-service-mesh/osm/pkg/service"
)

const (
	// ConnectionTimeout is the timeout duration used by Envoy to timeout connections
	ConnectionTimeout = 5 * time.Second

	// TransportProtocolTLS is the TLS transport protocol used in Envoy configurations
	TransportProtocolTLS = "tls"
)

// GetAddress creates an Envoy Address struct.
func GetAddress(address string, port uint32) *core.Address {
	// TODO(draychev): figure this out from the service
	return &core.Address{
		Address: &core.Address_SocketAddress{
			SocketAddress: &core.SocketAddress{
				Protocol: core.SocketAddress_TCP,
				Address:  address,
				PortSpecifier: &core.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
}

// GetTLSParams creates Envoy TlsParameters struct.
func GetTLSParams() *auth.TlsParameters {
	return &auth.TlsParameters{
		TlsMinimumProtocolVersion: auth.TlsParameters_TLSv1_2,
		TlsMaximumProtocolVersion: auth.TlsParameters_TLSv1_3,
	}
}

// GetAccessLog creates an Envoy AccessLog struct.
func GetAccessLog() []*envoy_config_filter_accesslog_v2.AccessLog {
	accessLog, err := ptypes.MarshalAny(getFileAccessLog())
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling AccessLog object")
		return nil
	}
	return []*envoy_config_filter_accesslog_v2.AccessLog{{
		Name: wellknown.FileAccessLog,
		ConfigType: &envoy_config_filter_accesslog_v2.AccessLog_TypedConfig{
			TypedConfig: accessLog,
		}},
	}
}

func getFileAccessLog() *accesslog.FileAccessLog {
	accessLogger := &accesslog.FileAccessLog{
		Path: accessLogPath,
		AccessLogFormat: &accesslog.FileAccessLog_JsonFormat{
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

func getCommonTLSContext(serviceName service.NamespacedService, mTLS bool, dir service.Direction) *auth.CommonTlsContext {
	var certType service.CertType

	// Define root cert type
	if mTLS {
		switch dir {
		case service.Outbound:
			certType = service.RootCertTypeForMTLSOutbound
		case service.Inbound:
			certType = service.RootCertTypeForMTLSInbound
		}
	} else {
		certType = service.RootCertTypeForHTTPS
	}

	return &auth.CommonTlsContext{
		TlsParams: GetTLSParams(),
		TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
			// Example ==> Name: "service-cert:NameSpaceHere/ServiceNameHere"
			Name: service.CertResource{
				Service:  serviceName,
				CertType: service.ServiceCertType,
			}.String(),
			SdsConfig: GetADSConfigSource(),
		}},
		ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
			ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
				// Example ==> Name: "root-cert<type>:NameSpaceHere/ServiceNameHere"
				Name: service.CertResource{
					Service:  serviceName,
					CertType: certType,
				}.String(),
				SdsConfig: GetADSConfigSource(),
			},
		},
	}
}

// MessageToAny converts from proto message to proto Any and returns an error if any
func MessageToAny(pb proto.Message) (*any.Any, error) {
	msg, err := ptypes.MarshalAny(pb)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// GetDownstreamTLSContext creates a downstream Envoy TLS Context
func GetDownstreamTLSContext(serviceName service.NamespacedService, mTLS bool) *auth.DownstreamTlsContext {
	tlsConfig := &auth.DownstreamTlsContext{
		CommonTlsContext: getCommonTLSContext(serviceName, mTLS, service.Inbound),
		// When RequireClientCertificate is enabled trusted CA certs must be provided via ValidationContextType
		RequireClientCertificate: &wrappers.BoolValue{Value: mTLS},
	}
	return tlsConfig
}

// GetUpstreamTLSContext creates an upstream Envoy TLS Context
func GetUpstreamTLSContext(serviceName service.NamespacedService) *auth.UpstreamTlsContext {
	tlsConfig := &auth.UpstreamTlsContext{
		CommonTlsContext: getCommonTLSContext(serviceName, true /* mTLS */, service.Outbound),

		// The Sni field is going to be used to do FilterChainMatch in getInboundInMeshFilterChain()
		// The "Sni" field below of an incoming request will be matched aganist a list of server names
		// in FilterChainMatch.ServerNames
		Sni: serviceName.GetCommonName().String(),
	}
	return tlsConfig
}

// GetServiceCluster creates an Envoy Cluster struct.
func GetServiceCluster(clusterName string, serviceName service.NamespacedService) (*xds.Cluster, error) {
	marshalledUpstreamTLSContext, err := MessageToAny(GetUpstreamTLSContext(serviceName))
	if err != nil {
		return nil, err
	}
	return &xds.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       ptypes.DurationProto(ConnectionTimeout),
		LbPolicy:             xds.Cluster_ROUND_ROBIN,
		ClusterDiscoveryType: &xds.Cluster_Type{Type: xds.Cluster_EDS},
		EdsClusterConfig:     &xds.Cluster_EdsClusterConfig{EdsConfig: GetADSConfigSource()},
		TransportSocket: &core.TransportSocket{
			Name: TransportSocketTLS,
			ConfigType: &core.TransportSocket_TypedConfig{
				TypedConfig: marshalledUpstreamTLSContext,
			},
		},
	}, nil
}

// GetADSConfigSource creates an Envoy ConfigSource struct.
func GetADSConfigSource() *core.ConfigSource {
	return &core.ConfigSource{
		ConfigSourceSpecifier: &core.ConfigSource_Ads{
			Ads: &core.AggregatedConfigSource{},
		},
	}
}
