package azure

import (
	"context"
	"time"

	r "github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	c "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/eapache/channels"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/mesh"
)

// newClient creates an Azure Client
func newClient(subscriptionID string, namespace string, azureAuthFile string, maxAuthRetryCount int, retryPause time.Duration, announceChan *channels.RingChannel, meshSpec mesh.SpecI, providerIdent string) mesh.ComputeProviderI {
	var authorizer autorest.Authorizer
	var err error
	if authorizer, err = getAuthorizerWithRetry(azureAuthFile, maxAuthRetryCount, retryPause); err != nil {
		glog.Fatal("Failed obtaining authentication token for Azure Resource Manager")
	}

	// TODO(draychev): The subscriptionID should be observed from the AzureResource (SMI)
	az := Client{
		namespace:         namespace,
		publicIPsClient:   n.NewPublicIPAddressesClient(subscriptionID),
		groupsClient:      r.NewGroupsClient(subscriptionID),
		deploymentsClient: r.NewDeploymentsClient(subscriptionID),
		vmssClient:        c.NewVirtualMachineScaleSetsClient(subscriptionID),
		vmClient:          c.NewVirtualMachinesClient(subscriptionID),
		netClient:         n.NewInterfacesClient(subscriptionID),
		subscriptionID:    subscriptionID,
		ctx:               context.Background(),
		authorizer:        authorizer,
		announceChan:      announceChan,
		mesh:              meshSpec,
		providerIdent:     providerIdent,
	}

	az.publicIPsClient.Authorizer = az.authorizer
	az.groupsClient.Authorizer = az.authorizer
	az.deploymentsClient.Authorizer = az.authorizer
	az.vmssClient.Authorizer = az.authorizer
	az.vmClient.Authorizer = az.authorizer
	az.netClient.Authorizer = az.authorizer

	/*
		// TODO(draychev): enable this when you come up with a way to probe ARM w/ minimal context
			if err = waitForAzureAuth(az, maxAuthRetryCount, retryPause); err != nil {
				glog.Fatal("Failed authenticating with Azure Resource Manager: ", err)
			}
	*/

	return az
}
