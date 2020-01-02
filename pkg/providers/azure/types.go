package azure

import (
	"context"

	"github.com/deislabs/smc/pkg/mesh"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/eapache/channels"
)

// Client is an Azure Client
type Client struct {
	namespace         string
	publicIPsClient   network.PublicIPAddressesClient
	groupsClient      resources.GroupsClient
	deploymentsClient resources.DeploymentsClient
	vmssClient        compute.VirtualMachineScaleSetsClient
	vmClient          compute.VirtualMachinesClient
	authorizer        autorest.Authorizer
	netClient         network.InterfacesClient
	resourceGroup     string
	subscriptionID    string
	ctx               context.Context
	announceChan      *channels.RingChannel
	meshSpec          mesh.SpecI
}
