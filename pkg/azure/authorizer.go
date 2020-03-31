package azure

import (
	"errors"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
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
func GetAuthorizerWithRetry(azureAuthFile string) (autorest.Authorizer, error) {
	var err error
	retryCount := 0
	for {
		// Fetch a new token
		_ = os.Setenv("AZURE_AUTH_LOCATION", azureAuthFile)
		// The line below requires env var AZURE_AUTH_LOCATION
		if authorizer, err := auth.NewAuthorizerFromFile(network.DefaultBaseURI); err == nil && authorizer != nil {
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
