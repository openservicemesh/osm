package cds

import (
	"context"
	"fmt"
	"testing"
	"time"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
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

	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return []service.MeshService{tests.BookbuyerService}, nil
	}))

	mockCatalog.EXPECT().ListOutboundServicesForIdentity(tests.BookbuyerServiceIdentity).Return([]service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service}).AnyTimes()
	mockCatalog.EXPECT().GetTargetPortToProtocolMappingForService(tests.BookbuyerService).Return(map[uint32]string{uint32(80): "protocol"}, nil)
	mockCatalog.EXPECT().GetEgressTrafficPolicy(tests.BookbuyerServiceIdentity).Return(nil, nil).AnyTimes()
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsEgressEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().IsTracingEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().GetTracingHost().Return(constants.DefaultTracingHost).AnyTimes()
	mockConfigurator.EXPECT().GetTracingPort().Return(constants.DefaultTracingPort).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{}).AnyTimes()
	mockCatalog.EXPECT().GetKubeController().Return(mockKubeController).AnyTimes()

	podlabels := map[string]string{
		tests.SelectorKey:                tests.BookbuyerServiceName,
		constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
	}

	newPod1 := tests.NewPodFixture(tests.Namespace, fmt.Sprintf("pod-1-%s", proxyUUID), tests.BookbuyerServiceAccountName, podlabels)
	newPod1.Annotations = map[string]string{
		constants.PrometheusScrapeAnnotation: "true",
	}
	_, err = kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &newPod1, metav1.CreateOptions{})
	assert.Nil(err)

	mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&newPod1})
	mockKubeController.EXPECT().IsMetricsEnabled(&newPod1).Return(true)

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

	HTTP2ProtocolOptions, err := envoy.GetHTTP2ProtocolOptions()
	assert.Nil(err)

	expectedLocalCluster := &xds_cluster.Cluster{
		TransportSocketMatches:        nil,
		Name:                          "default/bookbuyer-local",
		AltStatName:                   "default/bookbuyer-local",
		ClusterDiscoveryType:          &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STRICT_DNS},
		EdsClusterConfig:              nil,
		ConnectTimeout:                ptypes.DurationProto(1 * time.Second),
		RespectDnsTtl:                 true,
		DnsLookupFamily:               xds_cluster.Cluster_V4_ONLY,
		TypedExtensionProtocolOptions: HTTP2ProtocolOptions,
		LoadAssignment: &xds_endpoint.ClusterLoadAssignment{
			ClusterName: "default/bookbuyer-local",
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
											Address:  constants.WildcardIPAddr,
											PortSpecifier: &xds_core.SocketAddress_PortValue{
												PortValue: uint32(80),
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

	upstreamTLSProto, err := ptypes.MarshalAny(envoy.GetUpstreamTLSContext(tests.BookbuyerServiceIdentity, tests.BookstoreV1Service))
	require.Nil(err)

	expectedBookstoreV1Cluster := &xds_cluster.Cluster{
		TransportSocketMatches:        nil,
		Name:                          "default/bookstore-v1",
		AltStatName:                   "",
		ClusterDiscoveryType:          &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS},
		TypedExtensionProtocolOptions: HTTP2ProtocolOptions,
		EdsClusterConfig: &xds_cluster.Cluster_EdsClusterConfig{
			EdsConfig: &xds_core.ConfigSource{
				ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
					Ads: &xds_core.AggregatedConfigSource{},
				},
				ResourceApiVersion: xds_core.ApiVersion_V3,
			},
			ServiceName: "",
		},
		ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: string(envoy.TypeUpstreamTLSContext),
					Value:   upstreamTLSProto.Value,
				},
			},
		},
	}

	upstreamTLSProto, err = ptypes.MarshalAny(envoy.GetUpstreamTLSContext(tests.BookbuyerServiceIdentity, tests.BookstoreV2Service))
	require.Nil(err)
	expectedBookstoreV2Cluster := &xds_cluster.Cluster{
		TransportSocketMatches:        nil,
		Name:                          "default/bookstore-v2",
		AltStatName:                   "",
		ClusterDiscoveryType:          &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS},
		TypedExtensionProtocolOptions: HTTP2ProtocolOptions,
		EdsClusterConfig: &xds_cluster.Cluster_EdsClusterConfig{
			EdsConfig: &xds_core.ConfigSource{
				ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
					Ads: &xds_core.AggregatedConfigSource{},
				},
				ResourceApiVersion: xds_core.ApiVersion_V3,
			},
			ServiceName: "",
		},
		ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: string(envoy.TypeUpstreamTLSContext),
					Value:   upstreamTLSProto.Value,
				},
			},
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
		ConnectTimeout:         ptypes.DurationProto(1 * time.Second),
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
		"default/bookstore-v1",
		"default/bookstore-v2",
		"default/bookbuyer-local",
		"passthrough-outbound",
		"envoy-metrics-cluster",
		"envoy-tracing-cluster",
	}

	var foundClusters []string

	for _, a := range actualClusters {
		fmt.Println(a.Name)
		if a.Name == "default/bookbuyer-local" {
			assert.Truef(cmp.Equal(expectedLocalCluster, a, protocmp.Transform()), cmp.Diff(expectedLocalCluster, a, protocmp.Transform()))
			foundClusters = append(foundClusters, "default/bookbuyer-local")
			continue
		}
		if a.Name == "default/bookstore-v1" {
			assert.Truef(cmp.Equal(expectedBookstoreV1Cluster, a, protocmp.Transform()), cmp.Diff(expectedBookstoreV1Cluster, a, protocmp.Transform()))

			upstreamTLSContext := xds_auth.UpstreamTlsContext{}
			err = ptypes.UnmarshalAny(a.TransportSocket.GetTypedConfig(), &upstreamTLSContext)
			require.Nil(err)

			assert.Equal(expectedBookstoreV1TLSContext.CommonTlsContext.TlsParams, upstreamTLSContext.CommonTlsContext.TlsParams)
			assert.Equal("bookstore-v1.default.svc.cluster.local", upstreamTLSContext.Sni)
			assert.Nil(a.LoadAssignment) //ClusterLoadAssignment setting for non EDS clusters

			foundClusters = append(foundClusters, "default/bookstore-v1")
			continue
		}

		if a.Name == "default/bookstore-v2" {
			assert.Truef(cmp.Equal(expectedBookstoreV2Cluster, a, protocmp.Transform()), cmp.Diff(expectedBookstoreV1Cluster, a, protocmp.Transform()))

			upstreamTLSContext := xds_auth.UpstreamTlsContext{}
			err = ptypes.UnmarshalAny(a.TransportSocket.GetTypedConfig(), &upstreamTLSContext)
			require.Nil(err)

			assert.Equal(expectedBookstoreV2TLSContext.CommonTlsContext.TlsParams, upstreamTLSContext.CommonTlsContext.TlsParams)
			assert.Equal("bookstore-v2.default.svc.cluster.local", upstreamTLSContext.Sni)
			assert.Nil(a.LoadAssignment) //ClusterLoadAssignment setting for non EDS clusters

			foundClusters = append(foundClusters, "default/bookstore-v2")
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
	}))
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", uuid.New(), envoy.KindSidecar, tests.BookbuyerServiceAccountName, tests.Namespace))
	proxy, err := envoy.NewProxy(cn, "", nil)
	tassert.Nil(t, err)

	proxyIdentity, err := envoy.GetServiceIdentityFromProxyCertificate(cn)
	tassert.NoError(t, err)

	ctrl := gomock.NewController(t)
	meshCatalog := catalog.NewMockMeshCataloger(ctrl)
	meshCatalog.EXPECT().ListOutboundServicesForIdentity(proxyIdentity).Return(nil).AnyTimes()

	resp, err := NewResponse(meshCatalog, proxy, nil, nil, nil, proxyRegistry)
	tassert.Error(t, err)
	tassert.Nil(t, resp)
}

func TestNewResponseGetLocalServiceClusterError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockKubeController := k8s.NewMockController(ctrl)
	cfg := configurator.NewMockConfigurator(ctrl)
	meshCatalog := catalog.NewMockMeshCataloger(ctrl)

	svc := service.MeshService{
		Namespace: "ns",
		Name:      "svc",
	}

	proxyIdentity := identity.K8sServiceAccount{Name: "svcacc", Namespace: "ns"}.ToServiceIdentity()
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return []service.MeshService{svc}, nil
	}))
	cn := envoy.NewXDSCertCommonName(uuid.New(), envoy.KindSidecar, "svcacc", "ns")
	proxy, err := envoy.NewProxy(cn, "", nil)
	tassert.Nil(t, err)

	meshCatalog.EXPECT().ListOutboundServicesForIdentity(proxyIdentity).Return(nil).Times(1)
	meshCatalog.EXPECT().GetTargetPortToProtocolMappingForService(svc).Return(nil, errors.New("some error")).Times(1)
	meshCatalog.EXPECT().GetKubeController().Return(mockKubeController).AnyTimes()
	mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{})
	cfg.EXPECT().IsTracingEnabled().Return(false).Times(1)

	resp, err := NewResponse(meshCatalog, proxy, nil, cfg, nil, proxyRegistry)
	tassert.Error(t, err)
	tassert.Nil(t, resp)
}

func TestNewResponseGetEgressTrafficPolicyError(t *testing.T) {
	proxyIdentity := identity.K8sServiceAccount{Name: "svcacc", Namespace: "ns"}.ToServiceIdentity()
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}))
	cn := envoy.NewXDSCertCommonName(uuid.New(), envoy.KindSidecar, "svcacc", "ns")
	proxy, err := envoy.NewProxy(cn, "", nil)
	tassert.Nil(t, err)

	ctrl := gomock.NewController(t)
	meshCatalog := catalog.NewMockMeshCataloger(ctrl)
	mockKubeController := k8s.NewMockController(ctrl)
	cfg := configurator.NewMockConfigurator(ctrl)

	meshCatalog.EXPECT().ListOutboundServicesForIdentity(proxyIdentity).Return(nil).Times(1)
	meshCatalog.EXPECT().GetEgressTrafficPolicy(proxyIdentity).Return(nil, errors.New("some error")).Times(1)
	meshCatalog.EXPECT().GetKubeController().Return(mockKubeController).AnyTimes()
	mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{})
	cfg.EXPECT().IsEgressEnabled().Return(false).Times(1)
	cfg.EXPECT().IsTracingEnabled().Return(false).Times(1)

	resp, err := NewResponse(meshCatalog, proxy, nil, cfg, nil, proxyRegistry)
	tassert.NoError(t, err)
	tassert.Empty(t, resp)
}

func TestNewResponseGetEgressTrafficPolicyNotEmpty(t *testing.T) {
	proxyIdentity := identity.K8sServiceAccount{Name: "svcacc", Namespace: "ns"}.ToServiceIdentity()
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}))
	cn := envoy.NewXDSCertCommonName(uuid.New(), envoy.KindSidecar, "svcacc", "ns")
	proxy, err := envoy.NewProxy(cn, "", nil)
	tassert.Nil(t, err)

	ctrl := gomock.NewController(t)
	meshCatalog := catalog.NewMockMeshCataloger(ctrl)
	mockKubeController := k8s.NewMockController(ctrl)
	cfg := configurator.NewMockConfigurator(ctrl)
	meshCatalog.EXPECT().ListOutboundServicesForIdentity(proxyIdentity).Return(nil).Times(1)
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

	resp, err := NewResponse(meshCatalog, proxy, nil, cfg, nil, proxyRegistry)
	tassert.NoError(t, err)
	tassert.Len(t, resp, 1)
	tassert.Equal(t, resp[0].(*xds_cluster.Cluster).Name, "my-cluster")
}

func TestNewResponseForGateway(t *testing.T) {
	proxyIdentity := identity.K8sServiceAccount{Name: "gateway", Namespace: "osm-system"}.ToServiceIdentity()
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}))
	cn := envoy.NewXDSCertCommonName(uuid.New(), envoy.KindGateway, "gateway", "osm-system")
	proxy, err := envoy.NewProxy(cn, "", nil)
	tassert.Nil(t, err)

	ctrl := gomock.NewController(t)
	meshCatalog := catalog.NewMockMeshCataloger(ctrl)
	mockKubeController := k8s.NewMockController(ctrl)
	cfg := configurator.NewMockConfigurator(ctrl)
	meshCatalog.EXPECT().ListOutboundServicesForIdentity(proxyIdentity).Return([]service.MeshService{
		tests.BookbuyerService,
		tests.BookwarehouseService,
	}).AnyTimes()
	meshCatalog.EXPECT().GetKubeController().Return(mockKubeController).AnyTimes()
	mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{})
	cfg.EXPECT().IsEgressEnabled().Return(false).Times(1)
	cfg.EXPECT().IsTracingEnabled().Return(false).Times(1)
	cfg.EXPECT().IsPermissiveTrafficPolicyMode().Return(true).AnyTimes()

	resp, err := NewResponse(meshCatalog, proxy, nil, cfg, nil, proxyRegistry)
	tassert.NoError(t, err)
	tassert.Len(t, resp, 2)
	tassert.Equal(t, "default/bookbuyer", resp[0].(*xds_cluster.Cluster).Name)
	tassert.Equal(t, "default/bookwarehouse", resp[1].(*xds_cluster.Cluster).Name)
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
