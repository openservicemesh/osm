package sds

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	"github.com/deislabs/smc/pkg/catalog"
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
	panic("NotImplemented")
}
