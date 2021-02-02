package keyvault

import (
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("azure-keyvault-client")
)

type client struct {
	client        *keyvault.BaseClient
	vaultURL      string
	announcements chan announcements.Announcement
}
