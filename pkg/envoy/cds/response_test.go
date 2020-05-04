package cds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/ingress"
	"github.com/open-service-mesh/osm/pkg/smi"
)

func newMeshCatalog() catalog.MeshCataloger {
	meshSpec := smi.NewFakeMeshSpecClient()
	certManager := tresor.NewFakeCertManager()
	ingressMonitor := ingress.NewFakeIngressMonitor()
	stop := make(<-chan struct{})
	var endpointProviders []endpoint.Provider
	return catalog.NewMeshCatalog(meshSpec, certManager, ingressMonitor, stop, endpointProviders...)
}

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
			catalog := newMeshCatalog()
			svc := endpoint.NamespacedService{
				Namespace: "b",
				Service:   "c",
			}
			proxy := envoy.NewProxy("blah", svc, nil)
			meshSpec := smi.NewFakeMeshSpecClient()
			resp, err := NewResponse(context.Background(), catalog, meshSpec, proxy, nil)
			Expect(err).ToNot(HaveOccurred())

			expected := xds.DiscoveryResponse{
				VersionInfo: "",
				Resources: []*any.Any{{
					TypeUrl: "type.googleapis.com/envoy.api.v2.Cluster",
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

			expectedCluster := xds.Cluster{
				TransportSocketMatches: nil,
				Name:                   "envoy-admin-cluster",
				AltStatName:            "envoy-admin-cluster",
				ClusterDiscoveryType:   &xds.Cluster_Type{Type: xds.Cluster_STATIC},
				EdsClusterConfig:       nil,
				ConnectTimeout:         ptypes.DurationProto(connectionTimeout),
				LoadAssignment:         expectedClusterLoadAssignment,
			}

			cluster := xds.Cluster{}
			err = ptypes.UnmarshalAny(resp.Resources[0], &cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(cluster.LoadAssignment.ClusterName).To(Equal(expectedClusterLoadAssignment.ClusterName))
			Expect(len(cluster.LoadAssignment.Endpoints)).To(Equal(len(expectedClusterLoadAssignment.Endpoints)))
			Expect(cluster.LoadAssignment.Endpoints[0].LbEndpoints).To(Equal(expectedClusterLoadAssignment.Endpoints[0].LbEndpoints))
			Expect(cluster.LoadAssignment).To(Equal(expectedClusterLoadAssignment))
			Expect(cluster).To(Equal(expectedCluster))
		})
	})

})
