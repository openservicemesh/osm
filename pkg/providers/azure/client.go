package azure

import (
	r "github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	c "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/rs/zerolog/log"

	"github.com/open-service-mesh/osm/pkg/smi"
)

// NewProvider creates an Azure Client
func NewProvider(subscriptionID string, azureAuthFile string, stop chan struct{}, meshSpec smi.MeshSpec, azureResourceClient ResourceClient, providerIdent string) Client {
	var authorizer autorest.Authorizer
	var err error
	if authorizer, err = getAuthorizerWithRetry(azureAuthFile); err != nil {
		log.Fatal().Err(err).Msg("Failed obtaining authentication token for Azure Resource Manager")
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
		meshSpec:       meshSpec,
		providerID:     providerIdent,

		// AzureResource Client is needed here so the Azure EndpointsProvider can resolve a Kubernetes ServiceName
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
		// TODO(draychev): enable this when you come up with a way to probe ARM w/ minimal context
			if err = waitForAzureAuth(az, maxAuthRetryCount, retryPause); err != nil {
				log.Fatal().Err(err).Msg("Failed authenticating with Azure Resource Manager")
			}
	*/

	if err := az.run(stop); err != nil {
		log.Fatal().Err(err).Msgf("[%s] Could not start Azure EndpointsProvider client", packageName)
	}

	return az
}
