package envoy

import (
	"fmt"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	"github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/deislabs/smc/pkg/endpoint"
)

const (
	// ServiceCertPrefix is the prefix for the service certificate resource name. Example: "service-cert:webservice"
	ServiceCertPrefix = "service-cert"

	// RootCertPrefix is the prefix for the root certificate resource name. Example: "root-cert:webservice"
	RootCertPrefix = "root-cert"

	// Separator is the separator between the prefix and the name of the certificate.
	Separator = ":"
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
	accessLog, err := ptypes.MarshalAny(getFileAccessLog())
	if err != nil {
		glog.Error("[LDS] Could con construct AccessLog struct: ", err)
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

func getCommonTLSContext(serviceName endpoint.ServiceName) *auth.CommonTlsContext {
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
func GetDownstreamTLSContext(serviceName endpoint.ServiceName) *any.Any {
	tlsConfig := &auth.DownstreamTlsContext{
		CommonTlsContext: getCommonTLSContext(serviceName),

		// When RequireClientCertificate is enabled trusted CA certs must be provided via ValidationContextType
		RequireClientCertificate: &wrappers.BoolValue{Value: true},
	}

	tls, err := ptypes.MarshalAny(tlsConfig)
	if err != nil {
		glog.Error("[CDS] Error marshalling DownstreamTLS: ", err)
		return nil
	}
	return tls
}

// GetUpstreamTLSContext creates an upstream Envoy TLS Context.
func GetUpstreamTLSContext(serviceName endpoint.ServiceName) *any.Any {
	tlsConfig := &auth.UpstreamTlsContext{
		CommonTlsContext: getCommonTLSContext(serviceName),
		Sni:              string(serviceName),
	}

	tls, err := ptypes.MarshalAny(tlsConfig)
	if err != nil {
		glog.Error("[CDS] Error marshalling UpstreamTLS: ", err)
		return nil
	}
	return tls
}

// GetServiceCluster creates an Envoy Cluster struct.
func GetServiceCluster(clusterName string, serviceName endpoint.ServiceName) *xds.Cluster {
	return &xds.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
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
