package cds

import (
	"context"
	"fmt"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("CDS Response", func() {
	Context("Test cds.NewResponse", func() {
		It("Returns unique list of clusters for CDS", func() {
			kubeClient := testclient.NewSimpleClientset()
			smiClient := smi.NewFakeMeshSpecClient()

			proxyUUID := fmt.Sprintf("proxy-0-%s", uuid.New())
			podName := fmt.Sprintf("pod-0-%s", uuid.New())

			// The format of the CN matters
			xdsCertificate := certificate.CommonName(fmt.Sprintf("%s.%s.%s.foo.bar", proxyUUID, tests.BookbuyerServiceAccountName, tests.Namespace))
			proxy := envoy.NewProxy(xdsCertificate, nil)

			{
				// Create a pod to match the CN
				pod := tests.NewPodTestFixture(tests.Namespace, podName)
				pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID // This is what links the Pod and the Certificate
				_, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			{
				// Create a service for the pod created above
				selectors := map[string]string{
					// These need to match teh POD created above
					tests.SelectorKey: tests.SelectorValue,
				}
				// The serviceName must match the SMI
				service := tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, selectors)
				_, err := kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			resp, err := NewResponse(context.Background(), catalog.NewFakeMeshCatalog(kubeClient), smiClient, proxy, nil)
			Expect(err).ToNot(HaveOccurred())

			expected := xds.DiscoveryResponse{
				VersionInfo: "",
				Resources: []*any.Any{
					{
						TypeUrl: string(envoy.TypeCDS),
						Value:   []byte{10, 17, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 115, 116, 111, 114, 101, 26, 4, 10, 2, 26, 0, 34, 2, 8, 5, 194, 1, 219, 1, 10, 27, 101, 110, 118, 111, 121, 46, 116, 114, 97, 110, 115, 112, 111, 114, 116, 95, 115, 111, 99, 107, 101, 116, 115, 46, 116, 108, 115, 26, 187, 1, 10, 56, 116, 121, 112, 101, 46, 103, 111, 111, 103, 108, 101, 97, 112, 105, 115, 46, 99, 111, 109, 47, 101, 110, 118, 111, 121, 46, 97, 112, 105, 46, 118, 50, 46, 97, 117, 116, 104, 46, 85, 112, 115, 116, 114, 101, 97, 109, 84, 108, 115, 67, 111, 110, 116, 101, 120, 116, 18, 127, 10, 88, 10, 4, 8, 3, 16, 4, 50, 36, 10, 30, 115, 101, 114, 118, 105, 99, 101, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 18, 2, 26, 0, 58, 42, 10, 36, 114, 111, 111, 116, 45, 99, 101, 114, 116, 45, 102, 111, 114, 45, 109, 116, 108, 115, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 18, 2, 26, 0, 18, 35, 98, 111, 111, 107, 98, 117, 121, 101, 114, 46, 100, 101, 102, 97, 117, 108, 116, 46, 115, 118, 99, 46, 99, 108, 117, 115, 116, 101, 114, 46, 108, 111, 99, 97, 108, 16, 3},
					}, {
						TypeUrl: string(envoy.TypeCDS),
						Value:   []byte{10, 19, 101, 110, 118, 111, 121, 45, 97, 100, 109, 105, 110, 45, 99, 108, 117, 115, 116, 101, 114, 34, 2, 8, 1, 226, 1, 19, 101, 110, 118, 111, 121, 45, 97, 100, 109, 105, 110, 45, 99, 108, 117, 115, 116, 101, 114, 138, 2, 49, 10, 19, 101, 110, 118, 111, 121, 45, 97, 100, 109, 105, 110, 45, 99, 108, 117, 115, 116, 101, 114, 18, 26, 18, 24, 34, 2, 8, 100, 10, 18, 10, 16, 10, 14, 18, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 24, 152, 117, 16, 0},
					}},
				Canary:  false,
				TypeUrl: string(envoy.TypeCDS),
				Nonce:   "",
			}

			// There are to any.Any resources in the ClusterDiscoveryStruct (Clusters)
			Expect(len((*resp).Resources)).To(Equal(2))

			expectedClusters := []xds.Cluster{
				{
					Name: "default/bookstore",
					TransportSocket: &envoy_api_v2_core.TransportSocket{
						Name: "envoy.transport_sockets.tls",
						ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
							TypedConfig: &any.Any{
								TypeUrl: "type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
								Value:   []byte{10, 88, 10, 4, 8, 3, 16, 4, 50, 36, 10, 30, 115, 101, 114, 118, 105, 99, 101, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 18, 2, 26, 0, 58, 42, 10, 36, 114, 111, 111, 116, 45, 99, 101, 114, 116, 45, 102, 111, 114, 45, 109, 116, 108, 115, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 18, 2, 26, 0, 18, 35, 98, 111, 111, 107, 98, 117, 121, 101, 114, 46, 100, 101, 102, 97, 117, 108, 116, 46, 115, 118, 99, 46, 99, 108, 117, 115, 116, 101, 114, 46, 108, 111, 99, 97, 108},
							},
						},
					},
				},
				{
					Name: "envoy-admin-cluster",
				},
			}

			for clusterIdx, expectedCluster := range expectedClusters {
				// The first cluster is the route to the Bookstore
				// Second cluster is the Envoy Admin
				cluster := xds.Cluster{}
				err = ptypes.UnmarshalAny(resp.Resources[clusterIdx], &cluster)
				Expect(err).ToNot(HaveOccurred())
				Expect(cluster.Name).To(Equal(expectedCluster.Name))
				Expect(cluster.TransportSocket).To(Equal(expectedCluster.TransportSocket))
			}

			Expect((*resp).Resources[0]).To(Equal(expected.Resources[0]))
			Expect((*resp).Resources[1]).To(Equal(expected.Resources[1]))
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
								Value:   []byte{10, 88, 10, 4, 8, 3, 16, 4, 50, 36, 10, 30, 115, 101, 114, 118, 105, 99, 101, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 18, 2, 26, 0, 58, 42, 10, 36, 114, 111, 111, 116, 45, 99, 101, 114, 116, 45, 102, 111, 114, 45, 109, 116, 108, 115, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 98, 117, 121, 101, 114, 18, 2, 26, 0, 18, 35, 98, 111, 111, 107, 98, 117, 121, 101, 114, 46, 100, 101, 102, 97, 117, 108, 116, 46, 115, 118, 99, 46, 99, 108, 117, 115, 116, 101, 114, 46, 108, 111, 99, 97, 108},
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
								Name: fmt.Sprintf("%s%s%s", envoy.RootCertTypeForMTLS, envoy.Separator, "default/bookstore"),
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
