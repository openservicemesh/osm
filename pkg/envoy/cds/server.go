package cds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	"github.com/deislabs/smc/pkg/catalog"
)

const (
	maxConnections = 10000

	typeUrl    = "type.googleapis.com/envoy.api.v2.Cluster"
	serverName = "CDS"
)

// NewCDSServer creates a new CDS server
func NewCDSServer(catalog catalog.MeshCataloger) *Server {
	return &Server{
		catalog: catalog,
	}
}

// DeltaClusters implements xds.ClusterDiscoveryServiceServer
func (s *Server) DeltaClusters(xds.ClusterDiscoveryService_DeltaClustersServer) error {
	panic("NotImplemented")
}

// FetchClusters implements xds.ClusterDiscoveryServiceServer
func (s *Server) FetchClusters(ctx context.Context, discReq *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	panic("NotImplemented")
}
