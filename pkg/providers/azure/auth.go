// Package azure implements methods for working with Azure.
//
// This package contains common functionality for working with Azure
// as a compute provider.
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

// ErrUnableToObtainArmAuth is the error returned when GetAuthorizerWithRetry has not been able to authorize with ARM
var ErrUnableToObtainArmAuth = errors.New("unable to obtain ARM authorizer")

// GetAuthorizerWithRetry obtains an Azure Resource Manager authorizer.
func GetAuthorizerWithRetry(azureAuthFile string, baseURI string) (autorest.Authorizer, error) {
	retryCount := 0
	for {
		// TODO(draychev): move this to CLI argument
		_ = os.Setenv("AZURE_AUTH_LOCATION", azureAuthFile)
		// The line below requires env var AZURE_AUTH_LOCATION
		authorizer, err := auth.NewAuthorizerFromFile(baseURI)
		if err != nil {
			log.Error().Err(err).Msgf("Error creating an authorizer for URI %s", baseURI)
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
