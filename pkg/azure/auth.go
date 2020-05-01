package azure

import (
	"errors"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

const (
	maxAuthRetryCount = 10
	retryPause        = 10 * time.Second
)

var (
	// ErrUnableToObtainArmAuth is the error returned when GetAuthorizerWithRetry has not been able to authorize with ARM
	ErrUnableToObtainArmAuth = errors.New("unable to obtain ARM authorizer")

	// ErrNoAzureAuthFile is the error return when no Azure auth file has been provided
	ErrNoAzureAuthFile = errors.New("no azure auth file")
)

type authFunc func(baseURI string) (autorest.Authorizer, error)

// GetAuthorizerWithRetry obtains an Azure Resource Manager authorizer.
func GetAuthorizerWithRetry(azureAuthFile string, baseURI string) (autorest.Authorizer, error) {
	return getAuthorizerWithRetry(azureAuthFile, baseURI, auth.NewAuthorizerFromFile)
}

// GetAuthorizerWithRetryForKeyVault obtains an Azure Key Vault authorizer.
func GetAuthorizerWithRetryForKeyVault(azureAuthFile string, baseURI string) (autorest.Authorizer, error) {
	return getAuthorizerWithRetry(azureAuthFile, baseURI, auth.NewAuthorizerFromFile)
}

func getAuthorizerWithRetry(azureAuthFile string, baseURI string, authorizer authFunc) (autorest.Authorizer, error) {
	if azureAuthFile == "" {
		return nil, ErrNoAzureAuthFile
	}
	retryCount := 0
	for {
		// TODO(draychev): move this to CLI argument
		_ = os.Setenv("AZURE_AUTH_LOCATION", azureAuthFile)
		// The line below requires env var AZURE_AUTH_LOCATION
		authorizer, err := authorizer(baseURI)
		if err != nil {
			log.Error().Err(err).Msgf("Error creating an authorizer for URI %s from file %s", baseURI, azureAuthFile)
			log.Error().Msgf("AZURE_AUTH_LOCATION=%s", os.Getenv("AZURE_AUTH_LOCATION"))
			return authorizer, err
		}
		if err == nil && authorizer != nil {
			return authorizer, nil
		}

		if retryCount >= maxAuthRetryCount {
			log.Error().Err(err).Msgf("Tried %d times to get ARM authorization token", retryCount)
			return nil, ErrUnableToObtainArmAuth
		}
		retryCount++
		log.Error().Err(err).Msgf("Failed fetching authorization token for ARM. Will retry in %v", retryPause)
		time.Sleep(retryPause)
	}
}
