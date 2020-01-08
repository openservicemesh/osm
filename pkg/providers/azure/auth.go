package azure

import (
	"os"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang/glog"
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
			glog.Errorf("Tried %d times to authenticate with ARM; Error: %s", retryCount, err)
			return errUnableToObtainArmAuth
		}
		retryCount++
		glog.Errorf("Failed fetching config for App Gateway instance. Will retry in %v. Error: %s", retryPause, err)
		time.Sleep(retryPause)
	}
}
*/

func getAuthorizerWithRetry(azureAuthFile string, maxAuthRetryCount int, retryPause time.Duration) (autorest.Authorizer, error) {
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
			glog.Errorf("Tried %d times to get ARM authorization token; Error: %s", retryCount, err)
			return nil, errUnableToObtainArmAuth
		}
		retryCount++
		glog.Errorf("Failed fetching authorization token for ARM. Will retry in %v. Error: %s", retryPause, err)
		time.Sleep(retryPause)
	}
}
