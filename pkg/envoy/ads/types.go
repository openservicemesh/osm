package ads


import (
	"context"
	"reflect"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

//Server implements the Envoy xDS Aggregate Discovery Services
type Server struct {
	ctx         context.Context
	catalog     catalog.MeshCataloger
	meshSpec    smi.MeshSpec
	xdsHandlers map[envoy.TypeURI]func(context.Context, catalog.MeshCataloger, smi.MeshSpec, *envoy.Proxy, *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error)
}
