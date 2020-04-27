package envoy

import (
	"fmt"
	"time"

	"github.com/open-service-mesh/osm/pkg/constants"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	envoy_config_filter_accesslog_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/open-service-mesh/osm/pkg/endpoint"
)

const (
	// ServiceCertPrefix is the prefix for the service certificate resource name. Example: "service-cert:webservice"
	ServiceCertPrefix = "service-cert"

	// RootCertPrefix is the prefix for the root certificate resource name. Example: "root-cert:webservice"
	RootCertPrefix = "root-cert"

	// Separator is the separator between the prefix and the name of the certificate.
	Separator = ":"

	// ConnectionTimeout is the timeout duration used by Envoy to timeout connections
	ConnectionTimeout = 5 * time.Second

	// Cluster aggregating logs from all Envoy proxies
	logAggregator = constants.AggregatedDiscoveryServiceName
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
		CipherSuites:              []string{aes, chacha},
	}
}

// GetAccessLog creates an Envoy AccessLog struct.
func GetAccessLog() []*envoy_config_filter_accesslog_v2.AccessLog {
	return []*envoy_config_filter_accesslog_v2.AccessLog{
		getFileAccessLog(),

		// TODO(draychev): add feature flag
		getGRPCAccessLog(),
	}
}

func getFileAccessLog() *envoy_config_filter_accesslog_v2.AccessLog {
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

	accessLog, err := ptypes.MarshalAny(accessLogger)
	if err != nil {
		log.Error().Err(err).Msg("Error marshaling file access log Envoy config")
		return nil
	}

	return &envoy_config_filter_accesslog_v2.AccessLog{
		Name: wellknown.FileAccessLog,
		ConfigType: &envoy_config_filter_accesslog_v2.AccessLog_TypedConfig{
			TypedConfig: accessLog,
		},
	}
}

func getGRPCAccessLog() *envoy_config_filter_accesslog_v2.AccessLog {
	// Now GRPC log aggregator
	grpcAccessLogger := &accesslog.HttpGrpcAccessLogConfig{
		CommonConfig: &accesslog.CommonGrpcAccessLogConfig{
			GrpcService: &core.GrpcService{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
						ClusterName: logAggregator,
					},
				},
			},
		},
	}

	grpcAccessLog, err := ptypes.MarshalAny(grpcAccessLogger)
	if err != nil {
		log.Error().Err(err).Msg("Error marshaling file access log Envoy config")
		return nil
	}

	return &envoy_config_filter_accesslog_v2.AccessLog{
		Name: wellknown.HTTPGRPCAccessLog,
		ConfigType: &envoy_config_filter_accesslog_v2.AccessLog_TypedConfig{
			TypedConfig: grpcAccessLog,
		},
	}
}

func pbStringValue(v string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StringValue{
			StringValue: v,
		},
	}
}

func getCommonTLSContext(serviceName endpoint.NamespacedService) *auth.CommonTlsContext {
	return &auth.CommonTlsContext{
		TlsParams: GetTLSParams(),
		TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
			Name:      fmt.Sprintf("%s%s%s", ServiceCertPrefix, Separator, serviceName),
			SdsConfig: GetADSConfigSource(),
		}},
		ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
			ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
				Name:      fmt.Sprintf("%s%s%s", RootCertPrefix, Separator, serviceName),
				SdsConfig: GetADSConfigSource(),
			},
		},
	}
}

// GetDownstreamTLSContext creates a downstream Envoy TLS Context.
func GetDownstreamTLSContext(serviceName endpoint.NamespacedService) *any.Any {
	tlsConfig := &auth.DownstreamTlsContext{
		CommonTlsContext: getCommonTLSContext(serviceName),

		// When RequireClientCertificate is enabled trusted CA certs must be provided via ValidationContextType
		RequireClientCertificate: &wrappers.BoolValue{Value: true},
	}

	tls, err := ptypes.MarshalAny(tlsConfig)
	if err != nil {
		log.Error().Err(err).Msg("[CDS] Error marshalling DownstreamTLS")
		return nil
	}
	return tls
}

// GetUpstreamTLSContext creates an upstream Envoy TLS Context.
func GetUpstreamTLSContext(serviceName endpoint.NamespacedService) *any.Any {
	tlsConfig := &auth.UpstreamTlsContext{
		CommonTlsContext: getCommonTLSContext(serviceName),
		Sni:              serviceName.String(),
	}

	tls, err := ptypes.MarshalAny(tlsConfig)
	if err != nil {
		log.Error().Err(err).Msg("[CDS] Error marshalling UpstreamTLS")
		return nil
	}
	return tls
}

// GetServiceCluster creates an Envoy Cluster struct.
func GetServiceCluster(clusterName string, serviceName endpoint.NamespacedService) xds.Cluster {
	return xds.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       ptypes.DurationProto(ConnectionTimeout),
		LbPolicy:             xds.Cluster_ROUND_ROBIN,
		ClusterDiscoveryType: &xds.Cluster_Type{Type: xds.Cluster_EDS},
		EdsClusterConfig:     &xds.Cluster_EdsClusterConfig{EdsConfig: GetADSConfigSource()},
		TransportSocket: &core.TransportSocket{
			Name: TransportSocketTLS,
			ConfigType: &core.TransportSocket_TypedConfig{
				TypedConfig: GetUpstreamTLSContext(serviceName),
			},
		},
	}
}

// GetADSConfigSource creates an Envoy ConfigSource struct.
func GetADSConfigSource() *core.ConfigSource {
	return &core.ConfigSource{
		ConfigSourceSpecifier: &core.ConfigSource_Ads{
			Ads: &core.AggregatedConfigSource{},
		},
	}
}
