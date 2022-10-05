package bootstrap

import (
	"path/filepath"

	xds_accesslog_config "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	xds_bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_accesslog_stream "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	xds_transport_sockets "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_upstream_http "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/utils"
)

const (
	envoyTLSCertificateSecretName    = "tls_sds"
	envoyValidationContextSecretName = "validation_context_sds"

	// EnvoyBootstrapConfigFile is the name Envoy bootstrap configuration file
	EnvoyBootstrapConfigFile = "bootstrap.yaml"

	// EnvoyTLSCertificateSDSSecretFile is the name of the Envoy TLS certificate SDS config file
	EnvoyTLSCertificateSDSSecretFile = "tls_certificate_sds_secret.yaml"

	// EnvoyValidationContextSDSSecretFile is the name of the Envoy validation context SDS config file
	EnvoyValidationContextSDSSecretFile = "validation_context_sds_secret.yaml"

	// EnvoyProxyConfigPath is the path where the Envoy bootstrap config info is located
	EnvoyProxyConfigPath = "/etc/envoy"

	// EnvoyXDSCACertFile is the name of the Envoy XDS CA certificate file
	EnvoyXDSCACertFile = "cacert.pem"

	// EnvoyXDSCertFile is the name of the Envoy XDS certificate file
	EnvoyXDSCertFile = "sds_cert.pem"

	// EnvoyXDSKeyFile is the name of the Envoy XDS private key file
	EnvoyXDSKeyFile = "sds_key.pem"
)

var (
	envoyTLSCertificateConfigPath    = filepath.Join(EnvoyProxyConfigPath, EnvoyTLSCertificateSDSSecretFile)
	envoyValidationContextConfigPath = filepath.Join(EnvoyProxyConfigPath, EnvoyValidationContextSDSSecretFile)

	envoyXDSCertPath   = filepath.Join(EnvoyProxyConfigPath, EnvoyXDSCertFile)
	envoyXDSKeyPath    = filepath.Join(EnvoyProxyConfigPath, EnvoyXDSKeyFile)
	envoyXDSCACertPath = filepath.Join(EnvoyProxyConfigPath, EnvoyXDSCACertFile)
)

// BuildTLSSecret builds and returns an Envoy Discovery Response object for Envoy's xDS TLS
// Certificate
func BuildTLSSecret() (*xds_discovery.DiscoveryResponse, error) {
	secret := &xds_transport_sockets.Secret{
		Name: envoyTLSCertificateSecretName,
		Type: &xds_transport_sockets.Secret_TlsCertificate{
			TlsCertificate: &xds_transport_sockets.TlsCertificate{
				CertificateChain: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_Filename{
						Filename: envoyXDSCertPath,
					},
				},
				PrivateKey: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_Filename{
						Filename: envoyXDSKeyPath,
					},
				},
			},
		},
	}
	marshalledSecret, err := anypb.New(secret)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling Secret for Envoy's xDS TLS certificate resource")
		return nil, err
	}

	return &xds_discovery.DiscoveryResponse{
		Resources: []*any.Any{marshalledSecret},
	}, nil
}

// BuildValidationSecret builds and returns an Envoy Discovery Response object for Envoy's xDS
// Validation Context
func BuildValidationSecret() (*xds_discovery.DiscoveryResponse, error) {
	secret := &xds_transport_sockets.Secret{
		Name: envoyValidationContextSecretName,
		Type: &xds_transport_sockets.Secret_ValidationContext{
			ValidationContext: &xds_transport_sockets.CertificateValidationContext{
				TrustedCa: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_Filename{
						Filename: envoyXDSCACertPath,
					},
				},
			},
		},
	}
	marshalledSecret, err := anypb.New(secret)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling Secret for Envoy's xDS Validation Context resource")
		return nil, err
	}

	return &xds_discovery.DiscoveryResponse{
		Resources: []*any.Any{marshalledSecret},
	}, nil
}

// Build builds and returns an Envoy Bootstrap object from the given config
func (b *Builder) Build() (*xds_bootstrap.Bootstrap, error) {
	httpProtocolOptions := &xds_upstream_http.HttpProtocolOptions{
		UpstreamProtocolOptions: &xds_upstream_http.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &xds_upstream_http.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &xds_upstream_http.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{},
			},
		},
	}
	pbHTTPProtocolOptions, err := anypb.New(httpProtocolOptions)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshaling HttpProtocolOptions struct into an anypb.Any message")
		return nil, err
	}

	accessLogger := &xds_accesslog_stream.StdoutAccessLog{}
	pbAccessLog, err := anypb.New(accessLogger)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshaling StdoutAccessLog struct into an anypb.Any message")
		return nil, err
	}

	minVersionInt := xds_transport_sockets.TlsParameters_TlsProtocol_value[b.TLSMinProtocolVersion]
	maxVersionInt := xds_transport_sockets.TlsParameters_TlsProtocol_value[b.TLSMaxProtocolVersion]
	tlsMinVersion := xds_transport_sockets.TlsParameters_TlsProtocol(minVersionInt)
	tlsMaxVersion := xds_transport_sockets.TlsParameters_TlsProtocol(maxVersionInt)

	upstreamTLSContext := &xds_transport_sockets.UpstreamTlsContext{
		CommonTlsContext: &xds_transport_sockets.CommonTlsContext{
			AlpnProtocols: []string{
				"h2",
			},
			ValidationContextType: &xds_transport_sockets.CommonTlsContext_ValidationContextSdsSecretConfig{
				ValidationContextSdsSecretConfig: &xds_transport_sockets.SdsSecretConfig{
					Name: envoyValidationContextSecretName,
					SdsConfig: &xds_core.ConfigSource{
						ConfigSourceSpecifier: &xds_core.ConfigSource_Path{
							Path: envoyValidationContextConfigPath,
						},
					},
				},
			},
			TlsParams: &xds_transport_sockets.TlsParameters{
				TlsMinimumProtocolVersion: tlsMinVersion,
				TlsMaximumProtocolVersion: tlsMaxVersion,
				CipherSuites:              b.CipherSuites,
				EcdhCurves:                b.ECDHCurves,
			},
			TlsCertificateSdsSecretConfigs: []*xds_transport_sockets.SdsSecretConfig{
				{
					Name: envoyTLSCertificateSecretName,
					SdsConfig: &xds_core.ConfigSource{
						ConfigSourceSpecifier: &xds_core.ConfigSource_Path{
							Path: envoyTLSCertificateConfigPath,
						},
					},
				},
			},
		},
	}
	pbUpstreamTLSContext, err := anypb.New(upstreamTLSContext)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshaling UpstreamTlsContext struct into an anypb.Any message")
		return nil, err
	}

	bootstrap := &xds_bootstrap.Bootstrap{
		Node: &xds_core.Node{
			Id: b.NodeID,
		},
		Admin: &xds_bootstrap.Admin{
			AccessLog: []*xds_accesslog_config.AccessLog{
				{
					Name: envoy.AccessLoggerName,
					ConfigType: &xds_accesslog_config.AccessLog_TypedConfig{
						TypedConfig: pbAccessLog,
					},
				},
			},
			Address: &xds_core.Address{
				Address: &xds_core.Address_SocketAddress{
					SocketAddress: &xds_core.SocketAddress{
						Address: constants.LocalhostIPAddress,
						PortSpecifier: &xds_core.SocketAddress_PortValue{
							PortValue: constants.EnvoyAdminPort,
						},
					},
				},
			},
		},
		DynamicResources: &xds_bootstrap.Bootstrap_DynamicResources{
			AdsConfig: &xds_core.ApiConfigSource{
				ApiType:             xds_core.ApiConfigSource_GRPC,
				TransportApiVersion: xds_core.ApiVersion_V3,
				GrpcServices: []*xds_core.GrpcService{
					{
						TargetSpecifier: &xds_core.GrpcService_EnvoyGrpc_{
							EnvoyGrpc: &xds_core.GrpcService_EnvoyGrpc{
								ClusterName: constants.OSMControllerName,
							},
						},
					},
				},
				SetNodeOnFirstMessageOnly: true,
			},
			CdsConfig: &xds_core.ConfigSource{
				ResourceApiVersion: xds_core.ApiVersion_V3,
				ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
					Ads: &xds_core.AggregatedConfigSource{},
				},
			},
			LdsConfig: &xds_core.ConfigSource{
				ResourceApiVersion: xds_core.ApiVersion_V3,
				ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
					Ads: &xds_core.AggregatedConfigSource{},
				},
			},
		},
		StaticResources: &xds_bootstrap.Bootstrap_StaticResources{
			Clusters: []*xds_cluster.Cluster{
				{
					Name: constants.OSMControllerName,
					ClusterDiscoveryType: &xds_cluster.Cluster_Type{
						Type: xds_cluster.Cluster_LOGICAL_DNS,
					},
					TypedExtensionProtocolOptions: map[string]*any.Any{
						"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": pbHTTPProtocolOptions,
					},
					TransportSocket: &xds_core.TransportSocket{
						Name: "envoy.transport_sockets.tls",
						ConfigType: &xds_core.TransportSocket_TypedConfig{
							TypedConfig: pbUpstreamTLSContext,
						},
					},
					LbPolicy: xds_cluster.Cluster_ROUND_ROBIN,
					LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
						ClusterName: constants.OSMControllerName,
						Endpoints: []*xds_endpoint.LocalityLbEndpoints{
							{
								LbEndpoints: []*xds_endpoint.LbEndpoint{
									{
										HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
											Endpoint: &xds_endpoint.Endpoint{
												Address: &xds_core.Address{
													Address: &xds_core.Address_SocketAddress{
														SocketAddress: &xds_core.SocketAddress{
															Address: b.XDSHost,
															PortSpecifier: &xds_core.SocketAddress_PortValue{
																PortValue: constants.ADSServerPort,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					UpstreamConnectionOptions: &xds_cluster.UpstreamConnectionOptions{
						TcpKeepalive: &xds_core.TcpKeepalive{
							KeepaliveProbes:   wrapperspb.UInt32(5),
							KeepaliveTime:     wrapperspb.UInt32(60),
							KeepaliveInterval: wrapperspb.UInt32(5),
						},
					},
				},
			},
		},
	}

	probeListeners, probeClusters, err := b.getProbeResources()
	if err != nil {
		return nil, err
	}
	bootstrap.StaticResources.Listeners = append(bootstrap.StaticResources.Listeners, probeListeners...)
	bootstrap.StaticResources.Clusters = append(bootstrap.StaticResources.Clusters, probeClusters...)

	return bootstrap, nil
}

// GetTLSSDSConfigYAML returns the statically used TLS SDS config YAML.
func GetTLSSDSConfigYAML() ([]byte, error) {
	tlsSDSConfig, err := BuildTLSSecret()
	if err != nil {
		log.Error().Err(err).Msgf("Error building Envoy TLS Certificate SDS Config")
		return nil, err
	}

	configYAML, err := utils.ProtoToYAML(tlsSDSConfig)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingProtoToYAML)).
			Msgf("Failed to marshal Envoy TLS Certificate SDS Config to yaml")
		return nil, err
	}
	return configYAML, nil
}

// GetValidationContextSDSConfigYAML returns the statically used validation context SDS config YAML.
func GetValidationContextSDSConfigYAML() ([]byte, error) {
	validationContextSDSConfig, err := BuildValidationSecret()
	if err != nil {
		log.Error().Err(err).Msgf("Error building Envoy Validation Context SDS Config")
		return nil, err
	}

	configYAML, err := utils.ProtoToYAML(validationContextSDSConfig)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingProtoToYAML)).
			Msgf("Failed to marshal Envoy Validation Context SDS Config to yaml")
		return nil, err
	}
	return configYAML, nil
}

// getProbeResources returns the listener and cluster objects that are statically configured to serve
// startup, readiness and liveness probes.
// These will not change during the lifetime of the Pod.
// If the original probe defined a TCPSocket action, listener and cluster objects are not configured
// to serve that probe.
func (b *Builder) getProbeResources() ([]*xds_listener.Listener, []*xds_cluster.Cluster, error) {
	// This slice is the list of listeners for liveness, readiness, startup IF these have been configured in the Pod Spec
	var listeners []*xds_listener.Listener
	var clusters []*xds_cluster.Cluster

	// Is there a liveness probe in the Pod Spec?
	if b.OriginalHealthProbes.Liveness != nil && !b.OriginalHealthProbes.Liveness.IsTCPSocket {
		listener, err := getLivenessListener(b.OriginalHealthProbes.Liveness)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting liveness listener")
			return nil, nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getLivenessCluster(b.OriginalHealthProbes.Liveness))
	}

	// Is there a readiness probe in the Pod Spec?
	if b.OriginalHealthProbes.Readiness != nil && !b.OriginalHealthProbes.Readiness.IsTCPSocket {
		listener, err := getReadinessListener(b.OriginalHealthProbes.Readiness)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting readiness listener")
			return nil, nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getReadinessCluster(b.OriginalHealthProbes.Readiness))
	}

	// Is there a startup probe in the Pod Spec?
	if b.OriginalHealthProbes.Startup != nil && !b.OriginalHealthProbes.Startup.IsTCPSocket {
		listener, err := getStartupListener(b.OriginalHealthProbes.Startup)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting startup listener")
			return nil, nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getStartupCluster(b.OriginalHealthProbes.Startup))
	}

	return listeners, clusters, nil
}
