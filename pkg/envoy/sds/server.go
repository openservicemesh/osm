package sds

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	maxConnections = 10000
	typeUrl        = "type.googleapis.com/envoy.api.v2.auth.Secret"
	serverName     = "SDS"
)

// NewSDSServer creates a new SDS server
func NewSDSServer(catalog catalog.MeshCataloger) *Server {
	return &Server{
		connectionNum: 0,
		catalog:       catalog,
		closing:       make(chan bool),
	}
}

// DeltaSecrets implements sds.SecretDiscoveryServiceServer
func (s *Server) DeltaSecrets(xds.SecretDiscoveryService_DeltaSecretsServer) error {
	panic("NotImplemented")
}

// FetchSecrets implements sds.SecretDiscoveryServiceServer
func (s *Server) FetchSecrets(ctx context.Context, discReq *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	glog.Infof("[%s] Fetching Secrets...", serverName)

	cn, err := utils.ValidateClient(ctx, nil, serverName)
	if err != nil {
		glog.Errorf("[%s] Error constructing Secrets Discovery Response: %s", serverName, err)
		return nil, err
	}

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(ctx)
	proxy := envoy.NewProxy(cn, ip)
	s.catalog.RegisterProxy(proxy)

	glog.Infof("[%s][FetchSecrets] Responding to proxy %s", serverName, proxy.GetCommonName())
	return s.newDiscoveryResponse(proxy)
}
