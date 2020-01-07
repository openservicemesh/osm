package azure

import (
	"fmt"
	"strings"
	"time"

	"github.com/eapache/channels"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/mesh"
)

// NewProvider creates an Azure Client
func NewProvider(subscriptionID string, namespace string, azureAuthFile string, maxAuthRetryCount int, retryPause time.Duration, announceChan *channels.RingChannel, meshSpec mesh.SpecI, providerIdent string) mesh.ComputeProviderI {
	return newClient(subscriptionID, namespace, azureAuthFile, maxAuthRetryCount, retryPause, announceChan, meshSpec, providerIdent)
}

// GetIPs returns the IP addresses for the given ServiceName Name
// This function is required by the ComputeProviderI
func (az Client) GetIPs(svc mesh.ServiceName) []mesh.IP {
	var azureIPs []mesh.IP
	clusters := az.mesh.GetComputeIDForService(svc)
	for _, cluster := range clusters {
		if cluster.AzureID == "" {
			continue
		}

		glog.Infof("[azure] Getting IPs for service %s", svc)
		resourceGroup, kind, _, err := parseAzureID(cluster.AzureID)
		if err != nil {
			glog.Errorf("Unable to parse Azure URI %s: %s", cluster.AzureID, err)
			continue
		}

		var computeKindObserver = map[computeKind]computeObserver{
			vm:   az.getVM,
			vmss: az.getVMSS,
		}

		if observer, ok := computeKindObserver[kind]; ok {
			var ips []mesh.IP
			var err error
			ips, err = observer(resourceGroup, cluster.AzureID)
			if err != nil {
				glog.Error("Could not fetch VMSS services: ", err)
			}
			azureIPs = append(azureIPs, ips...)
		}
	}
	return azureIPs
}

// Run starts the Azure observer
func (az Client) Run(stopCh <-chan struct{}) error {
	glog.V(1).Infoln("Azure provider run started.")
	// TODO(draychev): implement pub/sub
	return nil
}

// GetID returns the unique identifier for the compute provider.
// This string will be used by logging tools to contextualize messages.
func (az Client) GetID() string {
	return az.providerIdent
}

func parseAzureID(id mesh.AzureID) (resourceGroup, computeKind, computeName, error) {
	// Sample URI: /resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz
	chunks := strings.Split(string(id), "/")
	if len(chunks) != 9 {
		return "", "", "", errIncorrectAzureURI
	}
	resGroup := resourceGroup(chunks[4])
	kind := computeKind(fmt.Sprintf("%s/%s", chunks[6], chunks[7]))
	name := computeName(chunks[8])
	return resGroup, kind, name, nil
}
