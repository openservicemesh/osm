package azure

import (
	r "github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	c "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/pkg/errors"

	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/providers/azure"
)

// NewProvider creates an Azure Client
func NewProvider(subscriptionID string, azureAuthFile string, stop chan struct{}, kubeController k8s.Controller, azureResourceClient ResourceClient, providerIdent string) (Client, error) {
	var authorizer autorest.Authorizer
	var err error
	var az Client

	if authorizer, err = azure.GetAuthorizerWithRetry(azureAuthFile, n.DefaultBaseURI); err != nil {
		return az, errors.Errorf("Failed to obtain authentication token for Azure Resource Manager: %+v", err)
	}

	az = Client{
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
		kubeController: kubeController,
		providerID:     providerIdent,

		// AzureResource Client is needed here so the Azure EndpointsProvider can resolve a Kubernetes service
		// into an Azure URI. (Example: resolve "webService" to an IP of a VM.)
		azureResourceClient: azureResourceClient,

		announcements: make(chan interface{}),
	}

	az.publicIPsClient.Authorizer = az.authorizer
	az.groupsClient.Authorizer = az.authorizer
	az.deploymentsClient.Authorizer = az.authorizer
	az.vmssClient.Authorizer = az.authorizer
	az.vmClient.Authorizer = az.authorizer
	az.netClient.Authorizer = az.authorizer

	/*
		// TODO(draychev): enable this when you find a way to probe ARM w/ minimal context
			if err = waitForAzureAuth(az, maxAuthRetryCount, retryPause); err != nil {
				log.Fatal().Err(err).Msg("Failed authenticating with Azure Resource Manager")
			}
	*/

	if err := az.run(stop); err != nil {
		return az, errors.Errorf("Failed to start Azure EndpointsProvider client: %+v", err)
	}

	return az, nil
}
