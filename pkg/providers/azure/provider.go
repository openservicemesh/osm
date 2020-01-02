package azure

import (
	"time"

	"github.com/eapache/channels"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/mesh"
	"github.com/deislabs/smc/pkg/mesh/providers"
)

// NewProvider creates an Azure Client
func NewProvider(subscriptionID string, resourceGroup string, namespace string, azureAuthFile string, maxAuthRetryCount int, retryPause time.Duration, announceChan *channels.RingChannel) mesh.ComputeProviderI {
	return newClient(subscriptionID, resourceGroup, namespace, azureAuthFile, maxAuthRetryCount, retryPause, announceChan)
}

// GetIPs returns the IP addresses for the given ServiceName Name; This is required by the ComputeProviderI
func (az Client) GetIPs(svc mesh.ServiceName) []mesh.IP {
	glog.Infof("[azure] Getting IPs for service %s", svc)
	az.meshSpec.GetComputeIDForService(svc, providers.Azure)

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

// Run starts the Azure observer
func (az Client) Run(stopCh <-chan struct{}) error {
	glog.V(1).Infoln("Azure provider run started.")
	// TODO(draychev): implement pub/sub
	return nil
}
