package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/eapache/channels"

	"github.com/deislabs/smc/pkg/mesh"
)

type resourceGroup string
type computeKind string
type computeName string

const (
	vm   computeKind = "Microsoft.Compute/virtualMachines"
	vmss computeKind = "Microsoft.Compute/virtualMachineScaleSets"
)

type computeObserver func(resourceGroup, mesh.AzureID) ([]mesh.IP, error)

// Client is an Azure Client
// Implements interfaces: ComputeProviderI
type Client struct {
	namespace         string
	publicIPsClient   network.PublicIPAddressesClient
	groupsClient      resources.GroupsClient
	deploymentsClient resources.DeploymentsClient
	vmssClient        compute.VirtualMachineScaleSetsClient
	vmClient          compute.VirtualMachinesClient
	authorizer        autorest.Authorizer
	netClient         network.InterfacesClient
	subscriptionID    string
	ctx               context.Context
	announceChan      *channels.RingChannel
	mesh              mesh.SpecI

	// Free-form string identifying the compute provider: Azure, Kubernetes etc.
	// This is used in logs
	providerIdent string
}
