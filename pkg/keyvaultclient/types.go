package keyvaultclient

import (
	"reflect"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/open-service-mesh/osm/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

type client struct {
	client        *keyvault.BaseClient
	vaultURL      string
	announcements chan interface{}
}
