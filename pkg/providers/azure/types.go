package azure

import (
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/eapache/channels"

	smc "github.com/deislabs/smc/pkg/apis/azureresource/v1"
	"github.com/deislabs/smc/pkg/mesh"
)

type resourceGroup string
type computeKind string
type computeName string

const (
	vm   computeKind = "Microsoft.Compute/virtualMachines"
	vmss computeKind = "Microsoft.Compute/virtualMachineScaleSets"
)

// azureID is a string type alias, which is the URI of a unique Azure cloud resource.
type azureID string

// computeObserver is a function which is specialized to a specific Azure compute and knows how to monitor this
// for IP address changes. For instance: VM, VMSS.
type computeObserver func(resourceGroup, azureID) ([]mesh.IP, error)

type azureClients struct {
	publicIPsClient network.PublicIPAddressesClient
	netClient       network.InterfacesClient

	vmClient   compute.VirtualMachinesClient
	vmssClient compute.VirtualMachineScaleSetsClient

	groupsClient      resources.GroupsClient
	deploymentsClient resources.DeploymentsClient

	authorizer autorest.Authorizer
}

// Client is an Azure Client struct. It implements EndpointsProvider interface
type Client struct {
	azureClients

	subscriptionID string
	announceChan   *channels.RingChannel
	meshTopology   mesh.Topology

	// Free-form string identifying the compute provider: Azure, Kubernetes etc.
	// This is used in logs
	providerID string

	// The AzureResource CRD client.
	// TODO(draychev): At this point we are deliberately making the choice to expose the SMI + Extensions storage mechanism
	// to the provider. At a later point we need to come up with an abstract mechanism, by which the Azure EndpointsProvider
	// will convert a service name to an Azure resource URI.
	azureResourceClient ResourceClient
}

// ResourceClient is an interface defining necessary functions to list the AzureResources.
type ResourceClient interface {
	// ListAzureResources lists the AzureResources, which will become mapping of a Service to an Azure URI.
	ListAzureResources() []*smc.AzureResource
}
