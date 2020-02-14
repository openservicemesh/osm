package ads

import (
	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy/cds"
	"github.com/deislabs/smc/pkg/envoy/eds"
	"github.com/deislabs/smc/pkg/envoy/lds"
	"github.com/deislabs/smc/pkg/envoy/rds"
	"github.com/deislabs/smc/pkg/envoy/sds"
)

//Server implements the Envoy xDS Aggregate Discovery Services
type Server struct {
	catalog     catalog.MeshCataloger
	lastVersion uint64
	lastNonce   string

	rdsServer *rds.Server
	edsServer *eds.Server
	ldsServer *lds.Server
	sdsServer *sds.Server
	cdsServer *cds.Server
}
