package envoy

import (
	"time"

	"github.com/golang/protobuf/ptypes/any"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	accessLogV2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
)

const (
	accessLogPath = "/dev/stdout"

	SDSClusterName = "sds"
	SDSAddress     = "sds.smc.svc.cluster.local"
	SDSPort        = 15123

	EDSClusterName = "eds"
	EDSAddress     = "eds.smc.svc.cluster.local"
	EDSPort        = 15124

	RDSClusterName = "rds"
	RDSAddress     = "rds.smc.svc.cluster.local"
	RDSPort        = 15126

	// cipher suites
	aes    = "ECDHE-ECDSA-AES128-GCM-SHA256"
	chacha = "ECDHE-ECDSA-CHACHA20-POLY1305"
)

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

func GetTLSParams() *auth.TlsParameters {
	return &auth.TlsParameters{
		TlsMinimumProtocolVersion: auth.TlsParameters_TLSv1_2,
		TlsMaximumProtocolVersion: auth.TlsParameters_TLSv1_3,
		CipherSuites:              []string{aes, chacha},
	}
}

func GetGRPCSource(clusterName string) *core.ConfigSource {
	return &core.ConfigSource{
		ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				ApiType: core.ApiConfigSource_GRPC,
				GrpcServices: []*core.GrpcService{{
					TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: clusterName},
					},
				}},
				SetNodeOnFirstMessageOnly: true,
			},
		},
	}
}

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

func GetWeightedCluster(clusterName string, weight uint32) *route.RouteAction_WeightedClusters {
	return &route.RouteAction_WeightedClusters{
		WeightedClusters: &route.WeightedCluster{
			Clusters: []*route.WeightedCluster_ClusterWeight{{
				Name: clusterName,
				Weight: &wrappers.UInt32Value{
					Value: weight,
				},
			}},
		},
	}
}

func GetUpstreamTLS(certPem string, keyPem string) *auth.UpstreamTlsContext {
	return &auth.UpstreamTlsContext{
		AllowRenegotiation: true,
		CommonTlsContext: &auth.CommonTlsContext{
			TlsParams: GetTLSParams(),
			TlsCertificates: []*auth.TlsCertificate{
				{
					CertificateChain: &core.DataSource{
						Specifier: &core.DataSource_Filename{
							Filename: certPem,
						},
					},
					PrivateKey: &core.DataSource{
						Specifier: &core.DataSource_Filename{
							Filename: keyPem,
						},
					},
				},
			},
		},
	}
}

func getTLS(clusterName string) *any.Any {
	tlsConfig := &auth.UpstreamTlsContext{
		CommonTlsContext: &auth.CommonTlsContext{
			TlsParams: GetTLSParams(),
			TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
				Name:      clusterName, // "bookstore.mesh"
				SdsConfig: GetGRPCSource(SDSClusterName),
			}},
			ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
				ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
					Name:      clusterName, // "bookstore.mesh"
					SdsConfig: GetGRPCSource(SDSClusterName),
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

func GetServiceCluster(clusterName string) *xds.Cluster {
	return &xds.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
		LbPolicy:             xds.Cluster_ROUND_ROBIN,
		ClusterDiscoveryType: &xds.Cluster_Type{Type: xds.Cluster_EDS},
		EdsClusterConfig:     GetEDSCluster(),
		TransportSocket: &core.TransportSocket{
			Name:       TransportSocketTLS,
			ConfigType: &core.TransportSocket_TypedConfig{TypedConfig: getTLS(clusterName)},
		},
	}
}

func GetEDSCluster() *xds.Cluster_EdsClusterConfig {
	return &xds.Cluster_EdsClusterConfig{
		EdsConfig: &core.ConfigSource{
			ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
				ApiConfigSource: &core.ApiConfigSource{
					ApiType: core.ApiConfigSource_GRPC,
					GrpcServices: []*core.GrpcService{
						{
							TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
								EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: EDSClusterName},
							},
						},
					},
				},
			},
		},
	}
}

func GetTimeout() *duration.Duration {
	return &duration.Duration{
		Seconds: 5,
	}
}

func GetH2() *core.Http2ProtocolOptions {
	return &core.Http2ProtocolOptions{}
}

func GetTransportSocket() *core.TransportSocket {
	tls, err := ptypes.MarshalAny(GetUpstreamTLS("/etc/ssl/certs/cert.pem", "/etc/ssl/certs/key.pem"))
	if err != nil {
		glog.Error("[CDS] Error marshalling UpstreamTLS: ", err)
		return nil
	}
	return &core.TransportSocket{
		Name: TransportSocketTLS,
		ConfigType: &core.TransportSocket_TypedConfig{
			TypedConfig: tls,
		},
	}
}

func GetLoadAssignment(clusterName string, address string, port uint32) *xds.ClusterLoadAssignment {
	return &xds.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{
			{
				LbEndpoints: []*endpoint.LbEndpoint{
					{
						HostIdentifier: &endpoint.LbEndpoint_Endpoint{
							Endpoint: &endpoint.Endpoint{
								Address: GetAddress(address, port),
							},
						},
					},
				},
			},
		},
	}
}
