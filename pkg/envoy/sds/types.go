package sds

import (
	"reflect"

	zlog "github.com/rs/zerolog/log"

	"github.com/open-service-mesh/osm/pkg/utils"
)

var (
	packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())
	log         = zlog.With().Str("comp", packageName).Caller().Logger()
)
