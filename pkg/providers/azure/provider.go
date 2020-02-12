package azure

import (
	"fmt"
	"net"
	"strings"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"

	smc "github.com/deislabs/smc/pkg/apis/azureresource/v1"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/log"
)

// ListEndpointsForService implements endpoints.Provider interface and returns the IP addresses and Ports for the given ServiceName Name.
func (az Client) ListEndpointsForService(svc endpoint.ServiceName) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint

	// TODO(draychev): resolve the actual port number of this service
	port := endpoint.Port(15003)
	var computeKindObserver = map[computeKind]computeObserver{
		vm:   az.getVM,
		vmss: az.getVMSS,
	}

	for _, azID := range az.resolveService(svc) {
		glog.Infof("[azure] Getting Endpoints for service %s", svc)
		resourceGroup, kind, _, err := parseAzureID(azID)
		if err != nil {
			glog.Errorf("[azure] Unable to parse Azure URI %s: %s", azID, err)
			continue
		}

		if observer, ok := computeKindObserver[kind]; ok {
			var ips []net.IP
			var err error
			ips, err = observer(resourceGroup, azID)
			if err != nil {
				glog.Error("[azure] Could not fetch VMSS services: ", err)
				continue
			}
			for _, ip := range ips {
				ept := endpoint.Endpoint{
					IP:   ip,
					Port: port,
				}
				endpoints = append(endpoints, ept)
			}
		}
	}
	return endpoints
}

// run starts the Azure observer
func (az Client) run(stop <-chan struct{}) error {
	glog.V(log.LvlInfo).Infoln("Azure provider run started.")
	// TODO(draychev): implement pub/sub
	return nil
}

func (az Client) GetAnnouncementsChannel() <-chan interface{} {
	return az.announcements
}

// GetID returns the unique identifier for the compute provider.
// This string will be used by logging tools to contextualize messages.
func (az Client) GetID() string {
	return az.providerID
}

func parseAzureID(id azureID) (resourceGroup, computeKind, computeName, error) {
	// Sample URI: /resource/subscriptions/e3f0/resourceGroups/meshSpec-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz
	chunks := strings.Split(string(id), "/")
	if len(chunks) != 9 {
		return "", "", "", errIncorrectAzureURI
	}
	resGroup := resourceGroup(chunks[4])
	kind := computeKind(fmt.Sprintf("%s/%s", chunks[6], chunks[7]))
	name := computeName(chunks[8])
	return resGroup, kind, name, nil
}

func (az *Client) resolveService(svc endpoint.ServiceName) []azureID {
	glog.V(log.LvlTrace).Infof("[azure] Resolving service %s to an Azure URI", svc)
	var azureIDs []azureID
	service, exists, err := az.meshSpec.GetService(svc)
	if err != nil {
		glog.Error("[azure] Error fetching Kubernetes Endpoints from cache: ", err)
		return azureIDs
	}
	if !exists {
		glog.Errorf("[azure] Error fetching Kubernetes Endpoints from cache: service %s does not exist", svc)
		return azureIDs
	}
	glog.Infof("[azure] Got the service: %+v", service)
	return matchServiceAzureResource(service, az.azureResourceClient.ListAzureResources())
}

type kv struct {
	k string
	v string
}

func matchServiceAzureResource(svc *corev1.Service, azureResourcesList []*smc.AzureResource) []azureID {
	glog.V(log.LvlTrace).Infof("[azure] Match service %s to an AzureID", svc)
	azureResources := make(map[kv]*smc.AzureResource)
	for _, azRes := range azureResourcesList {
		for k, v := range azRes.ObjectMeta.Labels {
			azureResources[kv{k, v}] = azRes
		}
	}
	uriSet := make(map[azureID]interface{})
	if service := svc; service != nil {
		for k, v := range service.ObjectMeta.Labels {
			if azRes, ok := azureResources[kv{k, v}]; ok && azRes != nil {
				uriSet[azureID(azRes.Spec.ResourceID)] = nil
			}
		}
	}
	// Ensure uniqueness
	var uris []azureID
	for uri := range uriSet {
		uris = append(uris, uri)
	}
	glog.V(log.LvlTrace).Infof("[azure] Found matches for service %s: %+v", svc, uris)
	return uris
}
