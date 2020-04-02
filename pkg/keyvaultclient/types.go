package keyvaultclient

import (
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/open-service-mesh/osm/pkg/logger"
)

var (
	log = logger.New("azure-keyvault-client")
)

type client struct {
	client        *keyvault.BaseClient
	vaultURL      string
	announcements chan interface{}
}
