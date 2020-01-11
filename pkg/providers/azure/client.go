package azure

import (
	r "github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	c "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/eapache/channels"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/mesh"
)

// NewProvider creates an Azure Client
func NewProvider(subscriptionID string, azureAuthFile string, announceChan *channels.RingChannel, stopChan chan struct{}, meshTopology mesh.Topology, azureResourceClient ResourceClient, providerIdent string) endpoint.Provider {
	var authorizer autorest.Authorizer
	var err error
	if authorizer, err = getAuthorizerWithRetry(azureAuthFile); err != nil {
		glog.Fatal("Failed obtaining authentication token for Azure Resource Manager")
	}

	// TODO(draychev): The subscriptionID should be observed from the AzureResource (SMI)
	az := Client{
		azureClients: azureClients{
			publicIPsClient: n.NewPublicIPAddressesClient(subscriptionID),
			netClient:       n.NewInterfacesClient(subscriptionID),

			groupsClient:      r.NewGroupsClient(subscriptionID),
			deploymentsClient: r.NewDeploymentsClient(subscriptionID),

			vmssClient: c.NewVirtualMachineScaleSetsClient(subscriptionID),
			vmClient:   c.NewVirtualMachinesClient(subscriptionID),

			authorizer: authorizer,
		},

		subscriptionID: subscriptionID,
		announceChan:   announceChan,
		meshTopology:   meshTopology,
		providerID:     providerIdent,

		// AzureResource Client is needed here so the Azure EndpointsProvider can resolve a Kubernetes ServiceName
		// into an Azure URI. (Example: resolve "webService" to an IP of a VM.)
		azureResourceClient: azureResourceClient,
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

	if err := az.Run(stopChan); err != nil {
		glog.Fatal("[azure] Could not start Azure EndpointsProvider client", err)
	}

	return az
}
