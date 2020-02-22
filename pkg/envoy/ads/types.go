package ads

import (
	"context"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/smi"
)

const (
	serverName = "ADS"
)

//Server implements the Envoy xDS Aggregate Discovery Services
type Server struct {
	ctx         context.Context
	catalog     catalog.MeshCataloger
	meshSpec    smi.MeshSpec
	xdsHandlers map[envoy.TypeURI]func(context.Context, catalog.MeshCataloger, smi.MeshSpec, *envoy.Proxy) (*envoy_api_v2.DiscoveryResponse, error)
}
