package ads

import (
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/cds"
	"github.com/deislabs/smc/pkg/envoy/eds"
	"github.com/deislabs/smc/pkg/envoy/lds"
	"github.com/deislabs/smc/pkg/envoy/rds"
	"github.com/deislabs/smc/pkg/envoy/sds"
)

// Server implements the Envoy xDS Aggregate Discovery Services
type Server struct {
	catalog catalog.MeshCataloger

	rdsServer *rds.Server
	edsServer *eds.Server
	ldsServer *lds.Server
	sdsServer *sds.Server
	cdsServer *cds.Server

	xdsHandlers map[envoy.TypeURI]func(*envoy.Proxy) (*envoy_api_v2.DiscoveryResponse, error)
}
