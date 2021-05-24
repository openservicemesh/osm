package injector

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xds_access "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	xds_bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_transport_sockets "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_upstream_http "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/version"
)

func getEnvoyConfigYAML(config envoyBootstrapConfigMeta, cfg configurator.Configurator) ([]byte, error) {
	bootstrap := &xds_bootstrap.Bootstrap{
		Node: &xds_core.Node{
			Id: config.NodeID,
		},
		Admin: &xds_bootstrap.Admin{
			AccessLog: []*xds_access.AccessLog{
				{
					Name: "envoy.access_loggers.stdout",
					ConfigType: &xds_access.AccessLog_TypedConfig{
						TypedConfig: &any.Any{
							TypeUrl: "type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog",
						},
					},
				},
			},
			Address: &xds_core.Address{
				Address: &xds_core.Address_SocketAddress{
					SocketAddress: &xds_core.SocketAddress{
						Address: constants.LocalhostIPAddress,
						PortSpecifier: &xds_core.SocketAddress_PortValue{
							PortValue: config.EnvoyAdminPort,
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
								ClusterName: config.XDSClusterName,
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
	}

	staticResources, err := getStaticResources(config)
	if err != nil {
		return nil, err
	}
	bootstrap.StaticResources = staticResources

	configYAML, err := protoToYAML(bootstrap)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to marshal envoy bootstrap config to yaml")
		return nil, err
	}
	return configYAML, nil
}

// getStaticResources returns STATIC resources included in the bootstrap Envoy config.
// These will not change during the lifetime of the Pod.
func getStaticResources(config envoyBootstrapConfigMeta) (*xds_bootstrap.Bootstrap_StaticResources, error) {
	// This slice is the list of listeners for liveness, readiness, startup IF these have been configured in the Pod Spec
	var listeners []*xds_listener.Listener

	var clusters []*xds_cluster.Cluster

	// There will ALWAYS be an xDS cluster
	xdsCluster, err := getXdsCluster(config)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting xDS cluster")
		return nil, err
	}
	clusters = append(clusters, xdsCluster)

	// Is there a liveness probe in the Pod Spec?
	if config.OriginalHealthProbes.liveness != nil {
		listener, err := getLivenessListener(config.OriginalHealthProbes.liveness)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting liveness listener")
			return nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getLivenessCluster(config.OriginalHealthProbes.liveness))
	}

	// Is there a readiness probe in the Pod Spec?
	if config.OriginalHealthProbes.readiness != nil {
		listener, err := getReadinessListener(config.OriginalHealthProbes.readiness)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting readiness listener")
			return nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getReadinessCluster(config.OriginalHealthProbes.readiness))
	}

	// Is there a startup probe in the Pod Spec?
	if config.OriginalHealthProbes.startup != nil {
		listener, err := getStartupListener(config.OriginalHealthProbes.startup)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting startup listener")
			return nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getStartupCluster(config.OriginalHealthProbes.startup))
	}

	return &xds_bootstrap.Bootstrap_StaticResources{
		Listeners: listeners,
		Clusters:  clusters,
	}, nil
}

func (wh *mutatingWebhook) createEnvoyBootstrapConfig(name, namespace, osmNamespace string, cert certificate.Certificater, originalHealthProbes healthProbes) (*corev1.Secret, error) {
	configMeta := envoyBootstrapConfigMeta{
		EnvoyAdminPort: constants.EnvoyAdminPort,
		XDSClusterName: constants.OSMControllerName,
		NodeID:         cert.GetCommonName().String(),

		RootCert: cert.GetIssuingCA(),
		Cert:     cert.GetCertificateChain(),
		Key:      cert.GetPrivateKey(),

		XDSHost: fmt.Sprintf("%s.%s.svc.cluster.local", constants.OSMControllerName, osmNamespace),
		XDSPort: constants.ADSServerPort,

		// OriginalHealthProbes stores the path and port for liveness, readiness, and startup health probes as initially
		// defined on the Pod Spec.
		OriginalHealthProbes: originalHealthProbes,
	}
	yamlContent, err := getEnvoyConfigYAML(configMeta, wh.configurator)
	if err != nil {
		log.Error().Err(err).Msg("Error creating Envoy bootstrap YAML")
		return nil, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
				constants.OSMAppInstanceLabelKey: wh.meshName,
				constants.OSMAppVersionLabelKey:  version.Version,
			},
		},
		Data: map[string][]byte{
			envoyBootstrapConfigFile: yamlContent,
		},
	}
	if existing, err := wh.kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{}); err == nil {
		log.Debug().Msgf("Updating bootstrap config Envoy: name=%s, namespace=%s", name, namespace)
		existing.Data = secret.Data
		return wh.kubeClient.CoreV1().Secrets(namespace).Update(context.Background(), existing, metav1.UpdateOptions{})
	}

	log.Debug().Msgf("Creating bootstrap config for Envoy: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
}

func getXdsCluster(config envoyBootstrapConfigMeta) (*xds_cluster.Cluster, error) {
	httpProtocolOptions := &xds_upstream_http.HttpProtocolOptions{
		UpstreamProtocolOptions: &xds_upstream_http.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &xds_upstream_http.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &xds_upstream_http.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{},
			},
		},
	}
	pbHTTPProtocolOptions, err := ptypes.MarshalAny(httpProtocolOptions)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling HttpProtocolOptions struct into an anypb.Any message")
		return nil, err
	}

	upstreamTLSContext := &xds_transport_sockets.UpstreamTlsContext{
		CommonTlsContext: &xds_transport_sockets.CommonTlsContext{
			AlpnProtocols: []string{
				"h2",
			},
			ValidationContextType: &xds_transport_sockets.CommonTlsContext_ValidationContext{
				ValidationContext: &xds_transport_sockets.CertificateValidationContext{
					TrustedCa: &xds_core.DataSource{
						Specifier: &xds_core.DataSource_InlineBytes{
							InlineBytes: config.RootCert,
						},
					},
				},
			},
			TlsParams: &xds_transport_sockets.TlsParameters{
				TlsMinimumProtocolVersion: xds_transport_sockets.TlsParameters_TLSv1_2,
				TlsMaximumProtocolVersion: xds_transport_sockets.TlsParameters_TLSv1_3,
			},
			TlsCertificates: []*xds_transport_sockets.TlsCertificate{
				{
					CertificateChain: &xds_core.DataSource{
						Specifier: &xds_core.DataSource_InlineBytes{
							InlineBytes: config.Cert,
						},
					},
					PrivateKey: &xds_core.DataSource{
						Specifier: &xds_core.DataSource_InlineBytes{
							InlineBytes: config.Key,
						},
					},
				},
			},
		},
	}
	pbUpstreamTLSContext, err := ptypes.MarshalAny(upstreamTLSContext)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling UpstreamTlsContext struct into an anypb.Any message")
		return nil, err
	}

	return &xds_cluster.Cluster{
		Name:           config.XDSClusterName,
		ConnectTimeout: durationpb.New(time.Millisecond * 250),
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
			ClusterName: config.XDSClusterName,
			Endpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*xds_endpoint.LbEndpoint{
						{
							HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
								Endpoint: &xds_endpoint.Endpoint{
									Address: &xds_core.Address{
										Address: &xds_core.Address_SocketAddress{
											SocketAddress: &xds_core.SocketAddress{
												Address: config.XDSHost,
												PortSpecifier: &xds_core.SocketAddress_PortValue{
													PortValue: config.XDSPort,
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
	}, nil
}

func protoToYAML(m protoreflect.ProtoMessage) ([]byte, error) {
	marshalOptions := protojson.MarshalOptions{
		UseProtoNames: true,
	}
	configJSON, err := marshalOptions.Marshal(m)
	if err != nil {
		return nil, err
	}

	configYAML, err := jsonToYAML(configJSON)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling xDS struct into YAML")
		return nil, err
	}
	return configYAML, err
}

// Reference impl taken from https://github.com/ghodss/yaml/blob/master/yaml.go#L87
func jsonToYAML(jb []byte) ([]byte, error) {
	// Convert the JSON to an object.
	var jsonObj interface{}
	// We are using yaml.Unmarshal here (instead of json.Unmarshal) because the
	// Go JSON library doesn't try to pick the right number type (int, float,
	// etc.) when unmarshalling to interface{}, it just picks float64
	// universally. go-yaml does go through the effort of picking the right
	// number type, so we can preserve number type throughout this process.
	err := yaml.Unmarshal([]byte(jb), &jsonObj)
	if err != nil {
		return nil, err
	}

	// Marshal this object into YAML.
	return yaml.Marshal(jsonObj)
}
