package injector

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	"github.com/openservicemesh/osm/pkg/envoy/bootstrap"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/utils"
	"github.com/openservicemesh/osm/pkg/version"
)

func getEnvoyConfigYAML(config envoyBootstrapConfigMeta, cfg configurator.Configurator) ([]byte, error) {
	bootstrapConfig, err := bootstrap.BuildFromConfig(bootstrap.Config{
		NodeID:           config.NodeID,
		AdminPort:        constants.EnvoyAdminPort,
		XDSClusterName:   constants.OSMControllerName,
		TrustedCA:        config.RootCert,
		CertificateChain: config.Cert,
		PrivateKey:       config.Key,
		XDSHost:          config.XDSHost,
		XDSPort:          config.XDSPort,
	})
	if err != nil {
		log.Error().Err(err).Msgf("Error building Envoy boostrap config")
		return nil, err
	}

	probeListeners, probeClusters, err := getProbeResources(config)
	if err != nil {
		return nil, err
	}
	bootstrapConfig.StaticResources.Listeners = append(bootstrapConfig.StaticResources.Listeners, probeListeners...)
	bootstrapConfig.StaticResources.Clusters = append(bootstrapConfig.StaticResources.Clusters, probeClusters...)

	configYAML, err := utils.ProtoToYAML(bootstrapConfig)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingProtoToYAML)).
			Msgf("Failed to marshal envoy bootstrap config to yaml")
		return nil, err
	}
	return configYAML, nil
}

// getProbeResources returns the listener and cluster objects that are statically configured to serve
// startup, readiness and liveness probes.
// These will not change during the lifetime of the Pod.
func getProbeResources(config envoyBootstrapConfigMeta) ([]*xds_listener.Listener, []*xds_cluster.Cluster, error) {
	// This slice is the list of listeners for liveness, readiness, startup IF these have been configured in the Pod Spec
	var listeners []*xds_listener.Listener
	var clusters []*xds_cluster.Cluster

	// Is there a liveness probe in the Pod Spec?
	if config.OriginalHealthProbes.liveness != nil {
		listener, err := getLivenessListener(config.OriginalHealthProbes.liveness)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting liveness listener")
			return nil, nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getLivenessCluster(config.OriginalHealthProbes.liveness))
	}

	// Is there a readiness probe in the Pod Spec?
	if config.OriginalHealthProbes.readiness != nil {
		listener, err := getReadinessListener(config.OriginalHealthProbes.readiness)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting readiness listener")
			return nil, nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getReadinessCluster(config.OriginalHealthProbes.readiness))
	}

	// Is there a startup probe in the Pod Spec?
	if config.OriginalHealthProbes.startup != nil {
		listener, err := getStartupListener(config.OriginalHealthProbes.startup)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting startup listener")
			return nil, nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getStartupCluster(config.OriginalHealthProbes.startup))
	}

	return listeners, clusters, nil
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
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshaling HttpProtocolOptions struct into an anypb.Any message")
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
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshaling UpstreamTlsContext struct into an anypb.Any message")
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
