package cds

import (
	"fmt"
	"testing"
	"time"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestNewResponse(t *testing.T) {
	assert := tassert.New(t)
	require := trequire.New(t)

	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)

	proxyUUID := uuid.New()
	// The format of the CN matters
	xdsCertificate := certificate.CommonName(fmt.Sprintf("%s.%s.%s.foo.bar", proxyUUID, tests.BookbuyerServiceAccountName, tests.Namespace))
	certSerialNumber := certificate.SerialNumber("123456")
	proxy := envoy.NewProxy(xdsCertificate, certSerialNumber, nil)

	mockCatalog.EXPECT().GetServicesFromEnvoyCertificate(xdsCertificate).Return([]service.MeshService{tests.BookbuyerService}, nil).AnyTimes()
	mockCatalog.EXPECT().ListAllowedOutboundServicesForIdentity(tests.BookbuyerServiceAccount).Return([]service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service}).AnyTimes()
	mockCatalog.EXPECT().GetTargetPortToProtocolMappingForService(tests.BookbuyerService).Return(map[uint32]string{uint32(80): "protocol"}, nil)
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsEgressEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().IsPrometheusScrapingEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().IsTracingEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().GetTracingHost().Return(constants.DefaultTracingHost).AnyTimes()
	mockConfigurator.EXPECT().GetTracingPort().Return(constants.DefaultTracingPort).AnyTimes()

	resp, err := NewResponse(mockCatalog, proxy, nil, mockConfigurator, nil)
	assert.Nil(err)

	// There are to any.Any resources in the ClusterDiscoveryStruct (Clusters)
	// There are 5 types of clusters that can exist based on the configuration:
	// 1. Destination cluster (Bookstore-v1, Bookstore-v2)
	// 2. Source cluster (Bookbuyer)
	// 3. Prometheus cluster
	// 4. Tracing cluster
	// 5. Passthrough cluster for egress
	numExpectedClusters := 6 // source and destination clusters
	assert.Equal(numExpectedClusters, len((*resp).Resources))
	actualClusters := []*xds_cluster.Cluster{}
	for _, resource := range (*resp).Resources {
		cl := xds_cluster.Cluster{}
		err = ptypes.UnmarshalAny(resource, &cl)
		require.Nil(err)
		actualClusters = append(actualClusters, &cl)
	}
	expectedLocalCluster := &xds_cluster.Cluster{
		TransportSocketMatches: nil,
		Name:                   "default/bookbuyer-local",
		AltStatName:            "default/bookbuyer-local",
		ClusterDiscoveryType:   &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STRICT_DNS},
		EdsClusterConfig:       nil,
		ConnectTimeout:         ptypes.DurationProto(1 * time.Second),
		ProtocolSelection:      xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
		RespectDnsTtl:          true,
		DnsLookupFamily:        xds_cluster.Cluster_V4_ONLY,
		Http2ProtocolOptions:   &xds_core.Http2ProtocolOptions{},
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

	upstreamTLSProto, err := ptypes.MarshalAny(envoy.GetUpstreamTLSContext(tests.BookbuyerServiceAccount, tests.BookstoreV1Service))
	require.Nil(err)

	expectedBookstoreV1Cluster := &xds_cluster.Cluster{
		TransportSocketMatches: nil,
		Name:                   "default/bookstore-v1",
		AltStatName:            "",
		ClusterDiscoveryType:   &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS},
		Http2ProtocolOptions:   &xds_core.Http2ProtocolOptions{},
		ProtocolSelection:      xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
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

	upstreamTLSProto, err = ptypes.MarshalAny(envoy.GetUpstreamTLSContext(tests.BookbuyerServiceAccount, tests.BookstoreV2Service))
	require.Nil(err)
	expectedBookstoreV2Cluster := &xds_cluster.Cluster{
		TransportSocketMatches: nil,
		Name:                   "default/bookstore-v2",
		AltStatName:            "",
		ClusterDiscoveryType:   &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS},
		Http2ProtocolOptions:   &xds_core.Http2ProtocolOptions{},
		ProtocolSelection:      xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
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
					Name: fmt.Sprintf("%s%s%s", envoy.RootCertTypeForMTLSOutbound, envoy.Separator, "default/bookstore-v1"),
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
					Name: fmt.Sprintf("%s%s%s", envoy.RootCertTypeForMTLSOutbound, envoy.Separator, "default/bookstore-v2"),
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

	foundClusters := []string{}

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
