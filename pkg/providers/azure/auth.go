package azure

import (
	"os"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/rs/zerolog/log"
)

const (
	maxAuthRetryCount = 10
	retryPause        = 10 * time.Second
)

/*
// TODO(draychev)
func waitForAzureAuth(azClient Client, maxAuthRetryCount int, retryPause time.Duration) error {
	retryCount := 0
	for {
		// TODO(draychev): this will not work -- come up with a different way to get a 200 OK
		_, err := azClient.getVMSS("someResourceGroup", "vmID") // Use any GET to verify auth
		if err == nil {
			return nil
		}

		if retryCount >= maxAuthRetryCount {
			log.Error().Err(err).Msgf("Tried %d times to authenticate with ARM", retryCount)
			return errUnableToObtainArmAuth
		}
		retryCount++
		log.Error().Err(err).Msgf("Failed fetching config for App Gateway instance, will retry in %v", retryPause)
		time.Sleep(retryPause)
	}
}
*/

func getAuthorizerWithRetry(azureAuthFile string) (autorest.Authorizer, error) {
	var err error
	retryCount := 0
	for {
		// Fetch a new token
		_ = os.Setenv("AZURE_AUTH_LOCATION", azureAuthFile)
		// The line below requires env var AZURE_AUTH_LOCATION
		if authorizer, err := auth.NewAuthorizerFromFile(n.DefaultBaseURI); err == nil && authorizer != nil {
			return authorizer, nil
		}

		if retryCount >= maxAuthRetryCount {
			log.Error().Err(err).Msgf("Tried %d times to get ARM authorization token", retryCount)
			return nil, errUnableToObtainArmAuth
		}
		retryCount++
		log.Error().Err(err).Msgf("Failed fetching authorization token for ARM. Will retry in %v", retryPause)
		time.Sleep(retryPause)
	}
}
