package cds

import (
	"context"
	"fmt"
	"testing"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestNewResponse(t *testing.T) {
	assert := tassert.New(t)
	require := trequire.New(t)

	mockCtrl := gomock.NewController(t)
	kubeClient := testclient.NewSimpleClientset()
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockKubeController := k8s.NewMockController(mockCtrl)

	proxyUUID := uuid.New()
	// The format of the CN matters
	xdsCertificate := envoy.NewXDSCertCommonName(proxyUUID, envoy.KindSidecar, tests.BookbuyerServiceAccountName, tests.Namespace)
	certSerialNumber := certificate.SerialNumber("123456")
	proxy, err := envoy.NewProxy(xdsCertificate, certSerialNumber, nil)
	assert.Nil(err)

	testMeshSvc := service.MeshService{
		Namespace:  tests.BookbuyerService.Namespace,
		Name:       tests.BookbuyerService.Namespace,
		Port:       80,
		TargetPort: 8080,
	}

	meshConfig := configv1alpha3.MeshConfig{
		Spec: configv1alpha3.MeshConfigSpec{
			Sidecar: configv1alpha3.SidecarSpec{
				TLSMinProtocolVersion: "TLSv1_2",
				TLSMaxProtocolVersion: "TLSv1_3",
			},
		},
	}

	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return []service.MeshService{testMeshSvc}, nil
	}), nil)

	expectedOutboundMeshPolicy := &trafficpolicy.OutboundMeshTrafficPolicy{
		ClustersConfigs: []*trafficpolicy.MeshClusterConfig{
			{
				Name:    "default/bookstore-v1|80",
				Service: tests.BookstoreV1Service,
			},
			{
				Name:    "default/bookstore-v2|80",
				Service: tests.BookstoreV2Service,
			},
		},
	}
	expectedInboundMeshPolicy := &trafficpolicy.InboundMeshTrafficPolicy{
		ClustersConfigs: []*trafficpolicy.MeshClusterConfig{
			{
				Name:    "default/bookbuyer|8080|local",
				Service: testMeshSvc,
				Port:    8080,
				Address: "127.0.0.1",
			},
		},
	}

	mockCatalog.EXPECT().GetInboundMeshTrafficPolicy(gomock.Any(), gomock.Any()).Return(expectedInboundMeshPolicy).AnyTimes()
	mockCatalog.EXPECT().GetOutboundMeshTrafficPolicy(tests.BookbuyerServiceIdentity).Return(expectedOutboundMeshPolicy).AnyTimes()
	mockCatalog.EXPECT().GetEgressTrafficPolicy(tests.BookbuyerServiceIdentity).Return(nil, nil).AnyTimes()
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsEgressEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().IsTracingEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().GetTracingHost().Return(constants.DefaultTracingHost).AnyTimes()
	mockConfigurator.EXPECT().GetTracingPort().Return(constants.DefaultTracingPort).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{}).AnyTimes()
	mockCatalog.EXPECT().GetKubeController().Return(mockKubeController).AnyTimes()
	mockConfigurator.EXPECT().GetMeshConfig().Return(meshConfig).AnyTimes()

	podlabels := map[string]string{
		constants.AppLabel:               testMeshSvc.Name,
		constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
	}

	newPod1 := tests.NewPodFixture(tests.Namespace, fmt.Sprintf("pod-1-%s", proxyUUID), tests.BookbuyerServiceAccountName, podlabels)
	newPod1.Annotations = map[string]string{
		constants.PrometheusScrapeAnnotation: "true",
	}
	_, err = kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &newPod1, metav1.CreateOptions{})
	assert.Nil(err)

	mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&newPod1})

	resp, err := NewResponse(mockCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
	assert.Nil(err)

	// There are to any.Any resources in the ClusterDiscoveryStruct (Clusters)
	// There are 5 types of clusters that can exist based on the configuration:
	// 1. Destination cluster (Bookstore-v1, Bookstore-v2)
	// 2. Source cluster (Bookbuyer)
	// 3. Prometheus cluster
	// 4. Tracing cluster
	// 5. Passthrough cluster for egress
	numExpectedClusters := 6 // source and destination clusters
	assert.Equal(numExpectedClusters, len(resp))
	var actualClusters []*xds_cluster.Cluster
	for idx := range resp {
		cl, ok := resp[idx].(*xds_cluster.Cluster)
		require.True(ok)
		actualClusters = append(actualClusters, cl)
	}

	typedHTTPProtocolOptions, err := getTypedHTTPProtocolOptions(getDefaultHTTPProtocolOptions())
	assert.Nil(err)

	expectedLocalCluster := &xds_cluster.Cluster{
		TransportSocketMatches:        nil,
		Name:                          "default/bookbuyer|8080|local",
		AltStatName:                   "default/bookbuyer|8080|local",
		ClusterDiscoveryType:          &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STRICT_DNS},
		EdsClusterConfig:              nil,
		RespectDnsTtl:                 true,
		DnsLookupFamily:               xds_cluster.Cluster_V4_ONLY,
		TypedExtensionProtocolOptions: typedHTTPProtocolOptions,
		LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
			ClusterName: "default/bookbuyer|8080|local",
			Endpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					Locality: &xds_core.Locality{
						Zone: "zone",
					},
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: &xds_core.Address{
									Address: &xds_core.Address_SocketAddress{
										SocketAddress: &xds_core.SocketAddress{
											Protocol: xds_core.SocketAddress_TCP,
											Address:  constants.LocalhostIPAddress,
											PortSpecifier: &xds_core.SocketAddress_PortValue{
												PortValue: uint32(8080),
											},
										},
									},
								},
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: constants.ClusterWeightAcceptAll,
						},
					}},
				},
			},
		},
	}

	upstreamTLSProto, err := anypb.New(envoy.GetUpstreamTLSContext(tests.BookbuyerServiceIdentity, tests.BookstoreV1Service, meshConfig.Spec.Sidecar))
	require.Nil(err)

	expectedBookstoreV1Cluster := &xds_cluster.Cluster{
		TransportSocketMatches:        nil,
		Name:                          "default/bookstore-v1|80",
		AltStatName:                   "",
		ClusterDiscoveryType:          &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS},
		TypedExtensionProtocolOptions: typedHTTPProtocolOptions,
		EdsClusterConfig: &xds_cluster.Cluster_EdsClusterConfig{
			EdsConfig: &xds_core.ConfigSource{
				ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
					Ads: &xds_core.AggregatedConfigSource{},
				},
				ResourceApiVersion: xds_core.ApiVersion_V3,
			},
			ServiceName: "",
		},
		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: string(envoy.TypeUpstreamTLSContext),
					Value:   upstreamTLSProto.Value,
				},
			},
		},
		CircuitBreakers: &xds_cluster.CircuitBreakers{
			Thresholds: []*xds_cluster.CircuitBreakers_Thresholds{getDefaultCircuitBreakerThreshold()},
		},
	}

	upstreamTLSProto, err = anypb.New(envoy.GetUpstreamTLSContext(tests.BookbuyerServiceIdentity, tests.BookstoreV2Service, meshConfig.Spec.Sidecar))
	require.Nil(err)
	expectedBookstoreV2Cluster := &xds_cluster.Cluster{
		TransportSocketMatches:        nil,
		Name:                          "default/bookstore-v2|80",
		AltStatName:                   "",
		ClusterDiscoveryType:          &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS},
		TypedExtensionProtocolOptions: typedHTTPProtocolOptions,
		EdsClusterConfig: &xds_cluster.Cluster_EdsClusterConfig{
			EdsConfig: &xds_core.ConfigSource{
				ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
					Ads: &xds_core.AggregatedConfigSource{},
				},
				ResourceApiVersion: xds_core.ApiVersion_V3,
			},
			ServiceName: "",
		},
		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: string(envoy.TypeUpstreamTLSContext),
					Value:   upstreamTLSProto.Value,
				},
			},
		},
		CircuitBreakers: &xds_cluster.CircuitBreakers{
			Thresholds: []*xds_cluster.CircuitBreakers_Thresholds{getDefaultCircuitBreakerThreshold()},
		},
	}

	expectedBookstoreV1TLSContext := xds_auth.UpstreamTlsContext{
		CommonTlsContext: &xds_auth.CommonTlsContext{
			TlsParams: &xds_auth.TlsParameters{
				TlsMinimumProtocolVersion: 3,
				TlsMaximumProtocolVersion: 4,
			},
			TlsCertificates: nil,
			TlsCertificateSdsSecretConfigs: []*xds_auth.SdsSecretConfig{{
				Name: "service-cert:default/bookstore-v1",
				SdsConfig: &xds_core.ConfigSource{
					ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
						Ads: &xds_core.AggregatedConfigSource{},
					},
				},
			}},
			ValidationContextType: &xds_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
				ValidationContextSdsSecretConfig: &xds_auth.SdsSecretConfig{
					Name: fmt.Sprintf("%s%s%s", secrets.RootCertTypeForMTLSOutbound, secrets.Separator, "default/bookstore-v1"),
					SdsConfig: &xds_core.ConfigSource{
						ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
							Ads: &xds_core.AggregatedConfigSource{},
						},
					},
				},
			},
			AlpnProtocols: envoy.ALPNInMesh,
		},
		Sni:                tests.BookstoreV1Service.ServerName(),
		AllowRenegotiation: false,
	}

	expectedBookstoreV2TLSContext := xds_auth.UpstreamTlsContext{
		CommonTlsContext: &xds_auth.CommonTlsContext{
			TlsParams: &xds_auth.TlsParameters{
				TlsMinimumProtocolVersion: 3,
				TlsMaximumProtocolVersion: 4,
			},
			TlsCertificates: nil,
			TlsCertificateSdsSecretConfigs: []*xds_auth.SdsSecretConfig{{
				Name: "service-cert:default/bookstore-v2",
				SdsConfig: &xds_core.ConfigSource{
					ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
						Ads: &xds_core.AggregatedConfigSource{},
					},
				},
			}},
			ValidationContextType: &xds_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
				ValidationContextSdsSecretConfig: &xds_auth.SdsSecretConfig{
					Name: fmt.Sprintf("%s%s%s", secrets.RootCertTypeForMTLSOutbound, secrets.Separator, "default/bookstore-v2"),
					SdsConfig: &xds_core.ConfigSource{
						ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
							Ads: &xds_core.AggregatedConfigSource{},
						},
					},
				},
			},
			AlpnProtocols: envoy.ALPNInMesh,
		},
		Sni:                tests.BookstoreV1Service.ServerName(),
		AllowRenegotiation: false,
	}

	expectedPrometheusCluster := &xds_cluster.Cluster{
		TransportSocketMatches: nil,
		Name:                   constants.EnvoyMetricsCluster,
		AltStatName:            constants.EnvoyMetricsCluster,
		ClusterDiscoveryType:   &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STATIC},
		EdsClusterConfig:       nil,
		LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
			ClusterName: constants.EnvoyMetricsCluster,
			Endpoints: []*xds_endpoint.LocalityLbEndpoints{
				{
					Locality: nil,
					LbEndpoints: []*xds_endpoint.LbEndpoint{{
						HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
							Endpoint: &xds_endpoint.Endpoint{
								Address: &xds_core.Address{
									Address: &xds_core.Address_SocketAddress{
										SocketAddress: &xds_core.SocketAddress{
											Protocol: xds_core.SocketAddress_TCP,
											Address:  "127.0.0.1",
											PortSpecifier: &xds_core.SocketAddress_PortValue{
												PortValue: uint32(15000),
											},
										},
									},
								},
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: 100,
						},
					}},
				},
			},
		},
	}

	expectedClusters := []string{
		"default/bookstore-v1|80",
		"default/bookstore-v2|80",
		"default/bookbuyer|8080|local",
		"passthrough-outbound",
		"envoy-metrics-cluster",
		"envoy-tracing-cluster",
	}

	var foundClusters []string

	for _, a := range actualClusters {
		if a.Name == "default/bookbuyer|8080|local" {
			assert.Truef(cmp.Equal(expectedLocalCluster, a, protocmp.Transform()), cmp.Diff(expectedLocalCluster, a, protocmp.Transform()))
			foundClusters = append(foundClusters, "default/bookbuyer|8080|local")
			continue
		}
		if a.Name == "default/bookstore-v1|80" {
			assert.Truef(cmp.Equal(expectedBookstoreV1Cluster, a, protocmp.Transform()), cmp.Diff(expectedBookstoreV1Cluster, a, protocmp.Transform()))

			upstreamTLSContext := xds_auth.UpstreamTlsContext{}
			err = a.TransportSocket.GetTypedConfig().UnmarshalTo(&upstreamTLSContext)
			require.Nil(err)

			assert.Equal(expectedBookstoreV1TLSContext.CommonTlsContext.TlsParams, upstreamTLSContext.CommonTlsContext.TlsParams)
			assert.Equal("bookstore-v1.default.svc.cluster.local", upstreamTLSContext.Sni)
			assert.Nil(a.LoadAssignment) //ClusterLoadAssignment setting for non EDS clusters

			foundClusters = append(foundClusters, "default/bookstore-v1|80")
			continue
		}

		if a.Name == "default/bookstore-v2|80" {
			assert.Truef(cmp.Equal(expectedBookstoreV2Cluster, a, protocmp.Transform()), cmp.Diff(expectedBookstoreV1Cluster, a, protocmp.Transform()))

			upstreamTLSContext := xds_auth.UpstreamTlsContext{}
			err = a.TransportSocket.GetTypedConfig().UnmarshalTo(&upstreamTLSContext)
			require.Nil(err)

			assert.Equal(expectedBookstoreV2TLSContext.CommonTlsContext.TlsParams, upstreamTLSContext.CommonTlsContext.TlsParams)
			assert.Equal("bookstore-v2.default.svc.cluster.local", upstreamTLSContext.Sni)
			assert.Nil(a.LoadAssignment) //ClusterLoadAssignment setting for non EDS clusters

			foundClusters = append(foundClusters, "default/bookstore-v2|80")
			continue
		}

		if a.Name == constants.EnvoyMetricsCluster {
			assert.Truef(cmp.Equal(expectedPrometheusCluster, a, protocmp.Transform()), cmp.Diff(expectedPrometheusCluster, a, protocmp.Transform()))

			foundClusters = append(foundClusters, constants.EnvoyMetricsCluster)
			continue
		}

		if a.Name == constants.EnvoyTracingCluster {
			foundClusters = append(foundClusters, constants.EnvoyTracingCluster)
			continue
		}

		if a.Name == envoy.OutboundPassthroughCluster {
			foundClusters = append(foundClusters, envoy.OutboundPassthroughCluster)
			continue
		}
	}
	assert.ElementsMatch(expectedClusters, foundClusters)
}

func TestNewResponseListServicesError(t *testing.T) {
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, errors.New("some error")
	}), nil)
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", uuid.New(), envoy.KindSidecar, tests.BookbuyerServiceAccountName, tests.Namespace))
	proxy, err := envoy.NewProxy(cn, "", nil)
	tassert.Nil(t, err)

	proxyIdentity, err := envoy.GetServiceIdentityFromProxyCertificate(cn)
	tassert.NoError(t, err)

	ctrl := gomock.NewController(t)
	meshCatalog := catalog.NewMockMeshCataloger(ctrl)
	cfg := configurator.NewMockConfigurator(ctrl)
	cfg.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()
	meshCatalog.EXPECT().GetOutboundMeshTrafficPolicy(proxyIdentity).Return(nil).AnyTimes()
	cfg.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

	resp, err := NewResponse(meshCatalog, proxy, nil, cfg, nil, proxyRegistry)
	tassert.Error(t, err)
	tassert.Nil(t, resp)
}

func TestNewResponseGetEgressTrafficPolicyError(t *testing.T) {
	proxyIdentity := identity.K8sServiceAccount{Name: "svcacc", Namespace: "ns"}.ToServiceIdentity()
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}), nil)
	cn := envoy.NewXDSCertCommonName(uuid.New(), envoy.KindSidecar, "svcacc", "ns")
	proxy, err := envoy.NewProxy(cn, "", nil)
	tassert.Nil(t, err)

	ctrl := gomock.NewController(t)
	meshCatalog := catalog.NewMockMeshCataloger(ctrl)
	mockKubeController := k8s.NewMockController(ctrl)
	cfg := configurator.NewMockConfigurator(ctrl)

	meshCatalog.EXPECT().GetInboundMeshTrafficPolicy(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	meshCatalog.EXPECT().GetOutboundMeshTrafficPolicy(proxyIdentity).Return(nil).Times(1)
	meshCatalog.EXPECT().GetEgressTrafficPolicy(proxyIdentity).Return(nil, errors.New("some error")).Times(1)
	meshCatalog.EXPECT().GetKubeController().Return(mockKubeController).AnyTimes()
	mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{})
	cfg.EXPECT().IsEgressEnabled().Return(false).Times(1)
	cfg.EXPECT().IsTracingEnabled().Return(false).Times(1)
	cfg.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()
	cfg.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

	resp, err := NewResponse(meshCatalog, proxy, nil, cfg, nil, proxyRegistry)
	tassert.NoError(t, err)
	tassert.Empty(t, resp)
}

func TestNewResponseGetEgressTrafficPolicyNotEmpty(t *testing.T) {
	proxyIdentity := identity.K8sServiceAccount{Name: "svcacc", Namespace: "ns"}.ToServiceIdentity()
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}), nil)
	cn := envoy.NewXDSCertCommonName(uuid.New(), envoy.KindSidecar, "svcacc", "ns")
	proxy, err := envoy.NewProxy(cn, "", nil)
	tassert.Nil(t, err)

	ctrl := gomock.NewController(t)
	meshCatalog := catalog.NewMockMeshCataloger(ctrl)
	mockKubeController := k8s.NewMockController(ctrl)
	cfg := configurator.NewMockConfigurator(ctrl)
	meshCatalog.EXPECT().GetInboundMeshTrafficPolicy(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	meshCatalog.EXPECT().GetOutboundMeshTrafficPolicy(proxyIdentity).Return(nil).Times(1)
	meshCatalog.EXPECT().GetKubeController().Return(mockKubeController).AnyTimes()
	mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{})
	meshCatalog.EXPECT().GetEgressTrafficPolicy(proxyIdentity).Return(&trafficpolicy.EgressTrafficPolicy{
		ClustersConfigs: []*trafficpolicy.EgressClusterConfig{
			{Name: "my-cluster"},
			{Name: "my-cluster"}, // the test ensures this duplicate is removed
		},
	}, nil).Times(1)
	cfg.EXPECT().IsEgressEnabled().Return(false).Times(1)
	cfg.EXPECT().IsTracingEnabled().Return(false).Times(1)
	cfg.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()
	cfg.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

	resp, err := NewResponse(meshCatalog, proxy, nil, cfg, nil, proxyRegistry)
	tassert.NoError(t, err)
	tassert.Len(t, resp, 1)
	tassert.Equal(t, resp[0].(*xds_cluster.Cluster).Name, "my-cluster")
}

func TestNewResponseForMulticlusterGateway(t *testing.T) {
	assert := tassert.New(t)

	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}), nil)
	cn := envoy.NewXDSCertCommonName(uuid.New(), envoy.KindGateway, "osm", "osm-system")
	proxy, err := envoy.NewProxy(cn, "", nil)
	assert.Nil(err)

	ctrl := gomock.NewController(t)
	meshCatalog := catalog.NewMockMeshCataloger(ctrl)
	cfg := configurator.NewMockConfigurator(ctrl)

	cfg.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()
	cfg.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	meshCatalog.EXPECT().ListOutboundServicesForMulticlusterGateway().Return([]service.MeshService{
		tests.BookstoreV1Service,
	}).AnyTimes()

	resp, err := NewResponse(meshCatalog, proxy, nil, cfg, nil, proxyRegistry)
	assert.NoError(err)
	assert.Len(resp, 1)
	assert.Equal(tests.BookstoreV1Service.ServerName(), resp[0].(*xds_cluster.Cluster).Name)
}

func TestRemoveDups(t *testing.T) {
	assert := tassert.New(t)

	orig := []*xds_cluster.Cluster{
		{
			Name: "c-1",
		},
		{
			Name: "c-2",
		},
		{
			Name: "c-1",
		},
	}
	assert.ElementsMatch([]types.Resource{
		&xds_cluster.Cluster{
			Name: "c-1",
		},
		&xds_cluster.Cluster{
			Name: "c-2",
		},
	}, removeDups(orig))
}
