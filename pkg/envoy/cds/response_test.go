package cds

import (
	"context"
	"fmt"
	"time"

	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("UniqueLists", func() {
	Context("Testing uniqueness of clusters", func() {
		It("Returns unique list of clusters for CDS", func() {

			// Create xds.cluster objects, some having the same cluster name
			clusters := []xds.Cluster{
				envoy.GetServiceCluster("osm/bookstore-1", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"}),
				envoy.GetServiceCluster("osm/bookstore-2", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"}),
				envoy.GetServiceCluster("osm/bookstore-1", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"}),
			}

			// Filter out xds.Cluster objects having the same name
			actualClusters := uniques(clusters)
			expectedClusters := []xds.Cluster{
				envoy.GetServiceCluster("osm/bookstore-1", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-1"}),
				envoy.GetServiceCluster("osm/bookstore-2", endpoint.NamespacedService{Namespace: "osm", Service: "bookstore-2"}),
			}

			Expect(actualClusters).To(Equal(expectedClusters))
		})
	})

	Context("Test cds.NewResponse", func() {
		It("Returns unique list of clusters for CDS", func() {
			cn := certificate.CommonName("bookbuyer.openservicemesh.io")
			proxy := envoy.NewProxy(cn, tests.BookbuyerService, nil)
			resp, err := NewResponse(context.Background(), catalog.NewFakeMeshCatalog(), smi.NewFakeMeshSpecClient(), proxy, nil)
			Expect(err).ToNot(HaveOccurred())

			expected := xds.DiscoveryResponse{
				VersionInfo: "",
				Resources: []*any.Any{
					{
						TypeUrl: string(envoy.TypeCDS),
						Value:   []byte{10, 17, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 115, 116, 111, 114, 101, 26, 4, 10, 2, 26, 0, 34, 2, 8, 5, 194, 1, 128, 2, 10, 27, 101, 110, 118, 111, 121, 46, 116, 114, 97, 110, 115, 112, 111, 114, 116, 95, 115, 111, 99, 107, 101, 116, 115, 46, 116, 108, 115, 26, 224, 1, 10, 56, 116, 121, 112, 101, 46, 103, 111, 111, 103, 108, 101, 97, 112, 105, 115, 46, 99, 111, 109, 47, 101, 110, 118, 111, 121, 46, 97, 112, 105, 46, 118, 50, 46, 97, 117, 116, 104, 46, 85, 112, 115, 116, 114, 101, 97, 109, 84, 108, 115, 67, 111, 110, 116, 101, 120, 116, 18, 163, 1, 10, 141, 1, 10, 66, 8, 3, 16, 4, 26, 29, 69, 67, 68, 72, 69, 45, 69, 67, 68, 83, 65, 45, 65, 69, 83, 49, 50, 56, 45, 71, 67, 77, 45, 83, 72, 65, 50, 53, 54, 26, 29, 69, 67, 68, 72, 69, 45, 69, 67, 68, 83, 65, 45, 67, 72, 65, 67, 72, 65, 50, 48, 45, 80, 79, 76, 89, 49, 51, 48, 53, 50, 36, 10, 30, 115, 101, 114, 118, 105, 99, 101, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 18, 2, 26, 0, 58, 33, 10, 27, 114, 111, 111, 116, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 18, 2, 26, 0, 18, 17, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 16, 3},
					}, {
						TypeUrl: string(envoy.TypeCDS),
						Value:   []byte{10, 19, 101, 110, 118, 111, 121, 45, 97, 100, 109, 105, 110, 45, 99, 108, 117, 115, 116, 101, 114, 34, 2, 8, 1, 226, 1, 19, 101, 110, 118, 111, 121, 45, 97, 100, 109, 105, 110, 45, 99, 108, 117, 115, 116, 101, 114, 138, 2, 49, 10, 19, 101, 110, 118, 111, 121, 45, 97, 100, 109, 105, 110, 45, 99, 108, 117, 115, 116, 101, 114, 18, 26, 18, 24, 34, 2, 8, 100, 10, 18, 10, 16, 10, 14, 18, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 24, 152, 117, 16, 0},
					}},
				Canary:  false,
				TypeUrl: string(envoy.TypeCDS),
				Nonce:   "",
			}
			Expect(*resp).To(Equal(expected))

			expectedClusterLoadAssignment := &xds.ClusterLoadAssignment{
				ClusterName: "envoy-admin-cluster",
				Endpoints: []*envoy_api_v2_endpoint.LocalityLbEndpoints{
					{
						Locality: nil,
						LbEndpoints: []*envoy_api_v2_endpoint.LbEndpoint{{
							HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
								Endpoint: &envoy_api_v2_endpoint.Endpoint{
									Address: &envoy_api_v2_core.Address{
										Address: &envoy_api_v2_core.Address_SocketAddress{
											SocketAddress: &envoy_api_v2_core.SocketAddress{
												Protocol: envoy_api_v2_core.SocketAddress_TCP,
												Address:  "127.0.0.1",
												PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
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
			}

			{
				expectedCluster := xds.Cluster{
					TransportSocketMatches: nil,
					Name:                   "default/bookstore",
					AltStatName:            "",
					ClusterDiscoveryType:   &xds.Cluster_Type{Type: xds.Cluster_EDS},
					EdsClusterConfig: &xds.Cluster_EdsClusterConfig{
						EdsConfig: &envoy_api_v2_core.ConfigSource{
							ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
								Ads: &envoy_api_v2_core.AggregatedConfigSource{},
							},
						},
						ServiceName: "",
					},
					ConnectTimeout: ptypes.DurationProto(5 * time.Second),
					TransportSocket: &envoy_api_v2_core.TransportSocket{
						Name: envoy.TransportSocketTLS,
						ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
							TypedConfig: &any.Any{
								TypeUrl: string(envoy.TypeUpstreamTLSContext),
								Value:   []byte{10, 141, 1, 10, 66, 8, 3, 16, 4, 26, 29, 69, 67, 68, 72, 69, 45, 69, 67, 68, 83, 65, 45, 65, 69, 83, 49, 50, 56, 45, 71, 67, 77, 45, 83, 72, 65, 50, 53, 54, 26, 29, 69, 67, 68, 72, 69, 45, 69, 67, 68, 83, 65, 45, 67, 72, 65, 67, 72, 65, 50, 48, 45, 80, 79, 76, 89, 49, 51, 48, 53, 50, 36, 10, 30, 115, 101, 114, 118, 105, 99, 101, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 18, 2, 26, 0, 58, 33, 10, 27, 114, 111, 111, 116, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 18, 2, 26, 0, 18, 17, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114},
							},
						},
					},
					LoadAssignment: expectedClusterLoadAssignment,
				}
				cluster := xds.Cluster{}
				err = ptypes.UnmarshalAny(resp.Resources[0], &cluster)
				Expect(err).ToNot(HaveOccurred())
				Expect(cluster.ClusterDiscoveryType).To(Equal(expectedCluster.ClusterDiscoveryType))
				Expect(cluster.EdsClusterConfig).To(Equal(expectedCluster.EdsClusterConfig))
				Expect(cluster.ConnectTimeout).To(Equal(expectedCluster.ConnectTimeout))
				Expect(cluster.TransportSocket).To(Equal(expectedCluster.TransportSocket))

				// TODO(draychev): finish the rest
				// Expect(cluster).To(Equal(expectedCluster))

				upstreamTLSContext := envoy_api_v2_auth.UpstreamTlsContext{}
				err = ptypes.UnmarshalAny(cluster.TransportSocket.GetTypedConfig(), &upstreamTLSContext)
				Expect(err).ToNot(HaveOccurred())

				expectedTLSContext := envoy_api_v2_auth.UpstreamTlsContext{
					CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
						TlsParams: &envoy_api_v2_auth.TlsParameters{
							TlsMinimumProtocolVersion: 3,
							TlsMaximumProtocolVersion: 4,
							CipherSuites: []string{
								"ECDHE-ECDSA-AES128-GCM-SHA256",
								"ECDHE-ECDSA-CHACHA20-POLY1305",
							},
						},
						TlsCertificates: nil,
						TlsCertificateSdsSecretConfigs: []*envoy_api_v2_auth.SdsSecretConfig{{
							Name: "service-cert:default/bookstore",
							SdsConfig: &envoy_api_v2_core.ConfigSource{
								ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
									Ads: &envoy_api_v2_core.AggregatedConfigSource{},
								},
							},
						}},
						ValidationContextType: &envoy_api_v2_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
							ValidationContextSdsSecretConfig: &envoy_api_v2_auth.SdsSecretConfig{
								Name: fmt.Sprintf("%s%s%s", envoy.RootCertPrefix, envoy.Separator, "default/bookstore"),
								SdsConfig: &envoy_api_v2_core.ConfigSource{
									ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
										Ads: &envoy_api_v2_core.AggregatedConfigSource{},
									},
								},
							},
						},
						AlpnProtocols: nil,
					},
					Sni:                "default/bookstore",
					AllowRenegotiation: false,
				}
				Expect(upstreamTLSContext.CommonTlsContext.TlsParams).To(Equal(expectedTLSContext.CommonTlsContext.TlsParams))
				// TODO(draychev): finish the rest
				// Expect(upstreamTLSContext).To(Equal(expectedTLSContext)
			}
			{
				expectedCluster := xds.Cluster{
					TransportSocketMatches: nil,
					Name:                   "envoy-admin-cluster",
					AltStatName:            "envoy-admin-cluster",
					ClusterDiscoveryType:   &xds.Cluster_Type{Type: xds.Cluster_STATIC},
					EdsClusterConfig:       nil,
					ConnectTimeout:         ptypes.DurationProto(1 * time.Second),
					LoadAssignment:         expectedClusterLoadAssignment,
				}
				cluster := xds.Cluster{}
				err = ptypes.UnmarshalAny(resp.Resources[1], &cluster)
				Expect(err).ToNot(HaveOccurred())
				Expect(cluster.LoadAssignment.ClusterName).To(Equal(expectedClusterLoadAssignment.ClusterName))
				Expect(len(cluster.LoadAssignment.Endpoints)).To(Equal(len(expectedClusterLoadAssignment.Endpoints)))
				Expect(cluster.LoadAssignment.Endpoints[0].LbEndpoints).To(Equal(expectedClusterLoadAssignment.Endpoints[0].LbEndpoints))
				Expect(cluster.LoadAssignment).To(Equal(expectedClusterLoadAssignment))
				Expect(cluster).To(Equal(expectedCluster))
			}
		})
	})

})
