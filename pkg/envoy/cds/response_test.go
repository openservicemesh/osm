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
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("CDS Response", func() {
	kubeClient := testclient.NewSimpleClientset()
	catalog := catalog.NewFakeMeshCatalog(kubeClient)
	proxyServiceName := tests.BookbuyerServiceName
	proxyServiceAccountName := tests.BookbuyerServiceAccountName
	proxyService := tests.BookbuyerService
	proxyServicePort := tests.ServicePort

	Context("Test cds.NewResponse", func() {
		It("Returns unique list of clusters for CDS", func() {
			smiClient := smi.NewFakeMeshSpecClient()

			proxyUUID := fmt.Sprintf("proxy-0-%s", uuid.New())
			podName := fmt.Sprintf("pod-0-%s", uuid.New())

			// The format of the CN matters
			xdsCertificate := certificate.CommonName(fmt.Sprintf("%s.%s.%s.foo.bar", proxyUUID, proxyServiceAccountName, tests.Namespace))
			proxy := envoy.NewProxy(xdsCertificate, nil)

			{
				// Create a pod to match the CN
				pod := tests.NewPodTestFixtureWithOptions(tests.Namespace, podName, proxyServiceAccountName)
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
				service := tests.NewServiceFixture(proxyServiceName, tests.Namespace, selectors)
				_, err := kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			config := &configurator.Config{
				OSMNamespace:     "-test-namespace-",
				EnablePrometheus: true,
				EnableTracing:    true,
			}
			resp, err := NewResponse(context.Background(), catalog, smiClient, proxy, nil, config)
			Expect(err).ToNot(HaveOccurred())

			// There are to any.Any resources in the ClusterDiscoveryStruct (Clusters)
			// There are 3 clusters we expect:
			// 1. Destination cluster
			// 2. Source cluster
			// 3. Prometheus cluster
			// 4. Zipkin cluster
			numExpectedClusters := 4
			Expect(len((*resp).Resources)).To(Equal(numExpectedClusters))
		})
	})

	Context("Test cds clusters", func() {
		It("Returns a local cluster object", func() {
			cluster, err := getServiceClusterLocal(catalog, proxyService, getLocalClusterName(proxyService))
			Expect(err).ToNot(HaveOccurred())

			expectedClusterLoadAssignment := &xds.ClusterLoadAssignment{
				ClusterName: getLocalClusterName(proxyService),
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
												Address:  constants.WildcardIPAddr,
												PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
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
			expectedCluster := xds.Cluster{
				TransportSocketMatches: nil,
				Name:                   getLocalClusterName(proxyService),
				AltStatName:            getLocalClusterName(proxyService),
				ClusterDiscoveryType:   &xds.Cluster_Type{Type: xds.Cluster_STATIC},
				EdsClusterConfig:       nil,
				ConnectTimeout:         ptypes.DurationProto(1 * time.Second),
				LoadAssignment:         expectedClusterLoadAssignment,
			}

			Expect(cluster.Name).To(Equal(expectedCluster.Name))
			Expect(cluster.LoadAssignment.ClusterName).To(Equal(expectedClusterLoadAssignment.ClusterName))
			Expect(len(cluster.LoadAssignment.Endpoints)).To(Equal(len(expectedClusterLoadAssignment.Endpoints)))
			Expect(cluster.LoadAssignment.Endpoints[0].LbEndpoints).To(Equal(expectedClusterLoadAssignment.Endpoints[0].LbEndpoints))
		})

		It("Returns a remote cluster object", func() {
			cluster, err := envoy.GetServiceCluster(proxyServiceName, proxyService)
			Expect(err).ToNot(HaveOccurred())

			expectedClusterLoadAssignment := &xds.ClusterLoadAssignment{
				ClusterName: constants.EnvoyMetricsCluster,
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
							Value:   []byte{},
						},
					},
				},
				LoadAssignment: expectedClusterLoadAssignment,
			}

			Expect(cluster.ClusterDiscoveryType).To(Equal(expectedCluster.ClusterDiscoveryType))
			Expect(cluster.EdsClusterConfig).To(Equal(expectedCluster.EdsClusterConfig))
			Expect(cluster.ConnectTimeout).To(Equal(expectedCluster.ConnectTimeout))
			// Not comparing the ever-chaning proto value, comparing rest of the struct though
			Expect(cluster.TransportSocket.GetName()).To(Equal(expectedCluster.TransportSocket.GetName()))
			Expect(cluster.TransportSocket.GetTypedConfig().GetTypeUrl()).To(Equal(expectedCluster.TransportSocket.GetTypedConfig().GetTypeUrl()))

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
							Name: fmt.Sprintf("%s%s%s", envoy.RootCertTypeForMTLSOutbound, envoy.Separator, "default/bookstore"),
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
		})

		It("Returns a Prometheus cluster object", func() {
			cluster := getPrometheusCluster()

			expectedClusterLoadAssignment := &xds.ClusterLoadAssignment{
				ClusterName: constants.EnvoyMetricsCluster,
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
			expectedCluster := xds.Cluster{
				TransportSocketMatches: nil,
				Name:                   constants.EnvoyMetricsCluster,
				AltStatName:            constants.EnvoyMetricsCluster,
				ClusterDiscoveryType:   &xds.Cluster_Type{Type: xds.Cluster_STATIC},
				EdsClusterConfig:       nil,
				ConnectTimeout:         ptypes.DurationProto(1 * time.Second),
				LoadAssignment:         expectedClusterLoadAssignment,
			}

			Expect(cluster.LoadAssignment.ClusterName).To(Equal(expectedClusterLoadAssignment.ClusterName))
			Expect(len(cluster.LoadAssignment.Endpoints)).To(Equal(len(expectedClusterLoadAssignment.Endpoints)))
			Expect(cluster.LoadAssignment.Endpoints[0].LbEndpoints).To(Equal(expectedClusterLoadAssignment.Endpoints[0].LbEndpoints))
			Expect(cluster.LoadAssignment).To(Equal(expectedClusterLoadAssignment))
			Expect(cluster).To(Equal(expectedCluster))
		})
	})
})
