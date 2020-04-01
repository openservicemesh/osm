package azure

import (
	"errors"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang/glog"
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
			glog.Errorf("Error creating an authorizer for URI %s: %s", baseURI, err)
			return authorizer, err
		}
		if err == nil && authorizer != nil {
			return authorizer, nil
		}

		if retryCount >= maxAuthRetryCount {
			glog.Errorf("Tried %d times to get ARM authorization token; Error: %s", retryCount, err)
			return nil, ErrUnableToObtainArmAuth
		}
		retryCount++
		glog.Errorf("Failed fetching authorization token for ARM. Will retry in %v. Error: %s", retryPause, err)
		time.Sleep(retryPause)
	}
}
