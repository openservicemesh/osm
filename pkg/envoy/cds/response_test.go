package cds

import (
	"context"
	"fmt"
	"time"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("CDS Response", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	kubeClient := testclient.NewSimpleClientset()
	catalog := catalog.NewFakeMeshCatalog(kubeClient)
	proxyServiceName := tests.BookbuyerServiceName
	proxyServiceAccountName := tests.BookbuyerServiceAccountName
	proxyService := tests.BookbuyerService
	proxyServicePort := tests.ServicePort

	Context("Test cds.NewResponse", func() {
		It("Returns unique list of clusters for CDS", func() {
			proxyUUID := uuid.New()
			podName := fmt.Sprintf("pod-0-%s", uuid.New())

			// The format of the CN matters
			xdsCertificate := certificate.CommonName(fmt.Sprintf("%s.%s.%s.foo.bar", proxyUUID, proxyServiceAccountName, tests.Namespace))
			certSerialNumber := certificate.SerialNumber("123456")
			proxy := envoy.NewProxy(xdsCertificate, certSerialNumber, nil)

			{
				// Create a pod to match the CN
				pod := tests.NewPodFixture(tests.Namespace, podName, proxyServiceAccountName, tests.PodLabels)
				pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String() // This is what links the Pod and the Certificate
				_, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			{
				// Create a service for the pod created above
				selectors := map[string]string{
					// These need to match the POD created above
					tests.SelectorKey: tests.SelectorValue,
				}
				// The serviceName must match the SMI
				service := tests.NewServiceFixture(proxyServiceName, tests.Namespace, selectors)
				_, err := kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
			mockConfigurator.EXPECT().IsPrometheusScrapingEnabled().Return(true).AnyTimes()
			mockConfigurator.EXPECT().IsTracingEnabled().Return(true).AnyTimes()
			mockConfigurator.EXPECT().IsEgressEnabled().Return(true).AnyTimes()
			mockConfigurator.EXPECT().GetTracingHost().Return(constants.DefaultTracingHost).AnyTimes()
			mockConfigurator.EXPECT().GetTracingPort().Return(constants.DefaultTracingPort).AnyTimes()

			resp, err := NewResponse(catalog, proxy, nil, mockConfigurator, nil)
			Expect(err).ToNot(HaveOccurred())

			// There are to any.Any resources in the ClusterDiscoveryStruct (Clusters)
			// There are 5 types of clusters that can exist based on the configuration:
			// 1. Destination cluster (Bookstore-v1, Bookstore-v2, and BookstoreApex)
			// 2. Source cluster (Bookbuyer)
			// 3. Prometheus cluster
			// 4. Tracing cluster
			// 5. Passthrough cluster for egress
			numExpectedClusters := 7 // source and destination clusters
			Expect(len((*resp).Resources)).To(Equal(numExpectedClusters))
		})
	})

	Context("Test cds clusters", func() {
		It("Returns a local cluster object", func() {
			localCluster, err := getLocalServiceCluster(catalog, proxyService, envoy.GetLocalClusterNameForService(proxyService))
			Expect(err).ToNot(HaveOccurred())

			expectedClusterLoadAssignment := &xds_endpoint.ClusterLoadAssignment{
				ClusterName: envoy.GetLocalClusterNameForService(proxyService),
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
												Address:  constants.WildcardIPAddr,
												PortSpecifier: &xds_core.SocketAddress_PortValue{
													PortValue: uint32(proxyServicePort),
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
			}

			expectedCluster := xds_cluster.Cluster{
				TransportSocketMatches: nil,
				Name:                   envoy.GetLocalClusterNameForService(proxyService),
				AltStatName:            envoy.GetLocalClusterNameForService(proxyService),
				ClusterDiscoveryType:   &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STATIC},
				EdsClusterConfig:       nil,
				ConnectTimeout:         ptypes.DurationProto(1 * time.Second),
				LoadAssignment:         expectedClusterLoadAssignment,
			}

			Expect(localCluster.Name).To(Equal(expectedCluster.Name))
			Expect(localCluster.LoadAssignment.ClusterName).To(Equal(expectedClusterLoadAssignment.ClusterName))
			Expect(len(localCluster.LoadAssignment.Endpoints)).To(Equal(len(expectedClusterLoadAssignment.Endpoints)))
			Expect(localCluster.LoadAssignment.Endpoints[0].LbEndpoints).To(Equal(expectedClusterLoadAssignment.Endpoints[0].LbEndpoints))
			Expect(localCluster.ProtocolSelection).To(Equal(xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL))
		})

		It("Returns a remote cluster object", func() {
			downstreamSvc := tests.BookbuyerService
			upstreamSvc := tests.BookstoreV1Service

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).Times(1)

			remoteCluster, err := getUpstreamServiceCluster(upstreamSvc, downstreamSvc, mockConfigurator)
			Expect(err).ToNot(HaveOccurred())

			expectedClusterLoadAssignment := &xds_endpoint.ClusterLoadAssignment{
				ClusterName: "test",
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
			}

			// Checking for the value by generating the same value the same way is redundant
			// Nonetheless, as getUpstreamServiceCluster logic gets more complicated, this might just be ok to have
			upstreamTLSProto, err := ptypes.MarshalAny(envoy.GetUpstreamTLSContext(proxyService, upstreamSvc))
			Expect(err).ToNot(HaveOccurred())

			expectedCluster := xds_cluster.Cluster{
				TransportSocketMatches: nil,
				Name:                   "default/bookstore",
				AltStatName:            "",
				ClusterDiscoveryType:   &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS},
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
				LoadAssignment: expectedClusterLoadAssignment,
			}

			Expect(remoteCluster.ClusterDiscoveryType).To(Equal(expectedCluster.ClusterDiscoveryType))
			Expect(remoteCluster.EdsClusterConfig).To(Equal(expectedCluster.EdsClusterConfig))
			Expect(remoteCluster.ConnectTimeout).To(Equal(expectedCluster.ConnectTimeout))
			Expect(remoteCluster.TransportSocket).To(Equal(expectedCluster.TransportSocket))

			// TODO(draychev): finish the rest
			// Expect(cluster).To(Equal(expectedCluster))

			upstreamTLSContext := xds_auth.UpstreamTlsContext{}
			err = ptypes.UnmarshalAny(remoteCluster.TransportSocket.GetTypedConfig(), &upstreamTLSContext)
			Expect(err).ToNot(HaveOccurred())

			expectedTLSContext := xds_auth.UpstreamTlsContext{
				CommonTlsContext: &xds_auth.CommonTlsContext{
					TlsParams: &xds_auth.TlsParameters{
						TlsMinimumProtocolVersion: 3,
						TlsMaximumProtocolVersion: 4,
					},
					TlsCertificates: nil,
					TlsCertificateSdsSecretConfigs: []*xds_auth.SdsSecretConfig{{
						Name: "service-cert:default/bookstore",
						SdsConfig: &xds_core.ConfigSource{
							ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
								Ads: &xds_core.AggregatedConfigSource{},
							},
						},
					}},
					ValidationContextType: &xds_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
						ValidationContextSdsSecretConfig: &xds_auth.SdsSecretConfig{
							Name: fmt.Sprintf("%s%s%s", envoy.RootCertTypeForMTLSOutbound, envoy.Separator, "default/bookstore"),
							SdsConfig: &xds_core.ConfigSource{
								ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
									Ads: &xds_core.AggregatedConfigSource{},
								},
							},
						},
					},
					AlpnProtocols: envoy.ALPNInMesh,
				},
				Sni:                upstreamSvc.ServerName(),
				AllowRenegotiation: false,
			}
			Expect(upstreamTLSContext.CommonTlsContext.TlsParams).To(Equal(expectedTLSContext.CommonTlsContext.TlsParams))
			Expect(upstreamTLSContext.Sni).To(Equal("bookstore-v1.default.svc.cluster.local"))
			// TODO(draychev): finish the rest
			// Expect(upstreamTLSContext).To(Equal(expectedTLSContext)
		})

		It("Returns a Prometheus cluster object", func() {
			remoteCluster := *getPrometheusCluster()

			expectedClusterLoadAssignment := &xds_endpoint.ClusterLoadAssignment{
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
			}
			expectedCluster := &xds_cluster.Cluster{
				TransportSocketMatches: nil,
				Name:                   constants.EnvoyMetricsCluster,
				AltStatName:            constants.EnvoyMetricsCluster,
				ClusterDiscoveryType:   &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_STATIC},
				EdsClusterConfig:       nil,
				ConnectTimeout:         ptypes.DurationProto(1 * time.Second),
				LoadAssignment:         expectedClusterLoadAssignment,
			}

			Expect(remoteCluster.LoadAssignment.ClusterName).To(Equal(expectedClusterLoadAssignment.ClusterName))
			Expect(len(remoteCluster.LoadAssignment.Endpoints)).To(Equal(len(expectedClusterLoadAssignment.Endpoints)))
			Expect(remoteCluster.LoadAssignment.Endpoints[0].LbEndpoints).To(Equal(expectedClusterLoadAssignment.Endpoints[0].LbEndpoints))
			Expect(remoteCluster.LoadAssignment).To(Equal(expectedClusterLoadAssignment))
			Expect(&remoteCluster).To(Equal(expectedCluster))
		})
	})
})
