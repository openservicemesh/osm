package envoy

import (
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	accessLogV2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	wellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	structpb "github.com/golang/protobuf/ptypes/struct"
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
func GetAccessLog() []*accessLogV2.AccessLog {
	accessLog, err := ptypes.MarshalAny(getFileAccessLog())
	if err != nil {
		glog.Error("[LDS] Could con construct AccessLog struct: ", err)
		return nil
	}
	return []*accessLogV2.AccessLog{
		{
			Name: wellknown.FileAccessLog,
			ConfigType: &accessLogV2.AccessLog_TypedConfig{
				TypedConfig: accessLog,
			},
		},
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

func getTLSDownstream(certificateName string) *any.Any {
	tlsConfig := &auth.DownstreamTlsContext{
		CommonTlsContext: &auth.CommonTlsContext{
			TlsParams: GetTLSParams(),
			TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
				Name:      certificateName,
				SdsConfig: GetADSConfigSource(),
			}},
			ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
				ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
					Name:      certificateName,
					SdsConfig: GetADSConfigSource(),
				},
			},
		},
	}

	tls, err := ptypes.MarshalAny(tlsConfig)
	if err != nil {
		glog.Error("[CDS] Error marshalling UpstreamTLS: ", err)
		return nil
	}
	return tls
}

func getTLSUpstream(certificateName string) *any.Any {
	tlsConfig := &auth.UpstreamTlsContext{
		CommonTlsContext: &auth.CommonTlsContext{
			TlsParams: GetTLSParams(),
			TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
				Name:      certificateName,
				SdsConfig: GetADSConfigSource(),
			}},
			ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
				ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
					Name:      certificateName,
					SdsConfig: GetADSConfigSource(),
				},
			},
		},
	}

	tls, err := ptypes.MarshalAny(tlsConfig)
	if err != nil {
		glog.Error("[CDS] Error marshalling UpstreamTLS: ", err)
		return nil
	}
	return tls
}

// GetTransportSocketForServiceDownstream creates a downstream Envoy TransportSocket struct.
func GetTransportSocketForServiceDownstream(certificateName string) *core.TransportSocket {
	return &core.TransportSocket{
		Name:       TransportSocketTLS,
		ConfigType: &core.TransportSocket_TypedConfig{TypedConfig: getTLSDownstream(certificateName)},
	}
}

// GetTransportSocketForServiceUpstream creates an upstream TransportSocket struct.
func GetTransportSocketForServiceUpstream(certificateName string) *core.TransportSocket {
	return &core.TransportSocket{
		Name:       TransportSocketTLS,
		ConfigType: &core.TransportSocket_TypedConfig{TypedConfig: getTLSUpstream(certificateName)},
	}
}

// GetServiceCluster creates an Envoy Cluster struct.
func GetServiceCluster(clusterName string, certificateName string) *xds.Cluster {
	return &xds.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
		LbPolicy:             xds.Cluster_ROUND_ROBIN,
		ClusterDiscoveryType: &xds.Cluster_Type{Type: xds.Cluster_EDS},
		EdsClusterConfig:     &xds.Cluster_EdsClusterConfig{EdsConfig: GetADSConfigSource()},
		TransportSocket:      GetTransportSocketForServiceUpstream(certificateName),
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
