package keyvaultclient

import (
	"reflect"

	zlog "github.com/rs/zerolog/log"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/open-service-mesh/osm/pkg/utils"
)

type empty struct{}

var (
	packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())
	log         = zlog.With().Str("comp", packageName).Caller().Logger()
)

type client struct {
	client        *keyvault.BaseClient
	vaultURL      string
	announcements chan interface{}
}
