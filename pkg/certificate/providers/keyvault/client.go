package keyvault

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	az "github.com/Azure/go-autorest/autorest/azure"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/providers/azure"
)

const (
	azureKeyVaultBaseURI   = "vault.azure.net"
	pollingDurationTimeout = 15 * time.Minute
)

func newKeyVaultClient(keyVaultName string, azureAuthFile string) (*client, error) {
	authorizer, err := azure.GetAuthorizerWithRetry(azureAuthFile, azureKeyVaultBaseURI)
	if err != nil {
		log.Error().Err(err).Msg("Error getting Azure Key Vault authorizer")
		return nil, err
	}

	keyVaultClient := keyvault.New()
	keyVaultClient.Authorizer = authorizer
	keyVaultClient.PollingDuration = pollingDurationTimeout
	return &client{
		client:        &keyVaultClient,
		vaultURL:      getKeyVaultURL(keyVaultName),
		announcements: make(chan announcements.Announcement),
	}, nil
}

func getKeyVaultURL(keyVaultName string) string {
	return fmt.Sprintf("https://%s.%s", keyVaultName, az.PublicCloud.KeyVaultDNSSuffix)
}
