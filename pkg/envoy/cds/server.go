package cds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	maxConnections = 10000

	typeUrl    = "type.googleapis.com/envoy.api.v2.Cluster"
	serverName = "CDS"
)

// NewCDSServer creates a new CDS server
func NewCDSServer(catalog catalog.ServiceCataloger) *Server {
	return &Server{
		connectionNum: 0,
		catalog:       catalog,
		closing:       make(chan bool),
	}
}

// DeltaClusters implements CDS.ClusterDiscoveryServiceServer
func (s *Server) DeltaClusters(xds.ClusterDiscoveryService_DeltaClustersServer) error {
	panic("NotImplemented")
}

// FetchClusters implements CDS.ClusterDiscoveryServiceServer
func (s *Server) FetchClusters(ctx context.Context, discReq *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Fetching Clusters...", serverName)

	cn, err := utils.ValidateClient(ctx, nil, serverName)
	if err != nil {
		glog.Errorf("[%s] Error constructing Clusters Discovery Response: %s", serverName, err)
		return nil, err
	}

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(ctx)
	proxy := envoy.NewProxy(cn, ip)
	// TODO(draychev):  s.catalog.RegisterProxy(proxy)

	glog.Infof("[%s][FetchClusters] Responding to proxy %s", serverName, proxy.GetCommonName())
	return s.newDiscoveryResponse(proxy)
}
