package azure

import (
	"context"
	"time"

	"github.com/eapache/channels"

	r "github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	c "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/mesh"
)

// NewProvider creates an Azure Client
func NewProvider(subscriptionID string, resourceGroup string, namespace string, azureAuthFile string, maxAuthRetryCount int, retryPause time.Duration, announceChan *channels.RingChannel) Client {
	var authorizer autorest.Authorizer
	var err error
	if authorizer, err = getAuthorizerWithRetry(azureAuthFile, maxAuthRetryCount, retryPause); err != nil {
		glog.Fatal("Failed obtaining authentication token for Azure Resource Manager")
	}
	az := Client{
		namespace:         namespace,
		publicIPsClient:   n.NewPublicIPAddressesClient(subscriptionID),
		groupsClient:      r.NewGroupsClient(subscriptionID),
		deploymentsClient: r.NewDeploymentsClient(subscriptionID),
		vmssClient:        c.NewVirtualMachineScaleSetsClient(subscriptionID),
		vmClient:          c.NewVirtualMachinesClient(subscriptionID),
		netClient:         n.NewInterfacesClient(subscriptionID),
		subscriptionID:    subscriptionID,
		resourceGroup:     resourceGroup,
		ctx:               context.Background(),
		authorizer:        authorizer,
		announceChan:      announceChan,
	}

	az.publicIPsClient.Authorizer = az.authorizer
	az.groupsClient.Authorizer = az.authorizer
	az.deploymentsClient.Authorizer = az.authorizer
	az.vmssClient.Authorizer = az.authorizer
	az.vmClient.Authorizer = az.authorizer
	az.netClient.Authorizer = az.authorizer

	if err = waitForAzureAuth(az, maxAuthRetryCount, retryPause); err != nil {
		glog.Fatal("Failed authenticating with Azure Resource Manager: ", err)
	}

	return az
}

// Run starts the Azure observer
func (az Client) Run(stopCh <-chan struct{}) error {
	glog.V(1).Infoln("Azure provider run started.")
	// TODO(draychev): implement pub/sub
	return nil
}

// GetIPs returns the IP addresses for the given ServiceName Name; This is required by the ComputeProviderI
func (az Client) GetIPs(svc mesh.ServiceName) []mesh.IP {
	glog.Infof("[azure] Getting IPs for service %s", svc)
	// TODO(draychev): from the ServiceName determine the AzureResource

	var ips []mesh.IP

	if vmssServices, err := az.getVMSS(); err != nil {
		glog.Error("Could not fetch VMSS services: ", err)
	} else {
		if vmssIPs, exists := vmssServices[svc]; exists {
			ips = append(ips, vmssIPs...)
		}
	}

	if vmServices, err := az.getVM(); err != nil {
		glog.Error("Could not fetch VM services: ", err)
	} else {
		if vmIPs, exists := vmServices[svc]; exists {
			ips = append(ips, vmIPs...)
		}
	}

	return ips
}
