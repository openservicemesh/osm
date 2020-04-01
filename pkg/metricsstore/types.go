package metricsstore

import (
	"reflect"

	"github.com/open-service-mesh/osm/pkg/logger"
	"github.com/open-service-mesh/osm/pkg/utils"
)

type empty struct{}

var (
	packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())
	log         = logger.New(packageName)
)
