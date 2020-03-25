package azure

import (
	"fmt"
	"net"
	"strings"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"

	osm "github.com/open-service-mesh/osm/pkg/apis/azureresource/v1"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/log/level"
)

// ListEndpointsForService implements endpoints.Provider interface and returns the IP addresses and Ports for the given ServiceName Name.
func (az Client) ListEndpointsForService(svc endpoint.ServiceName) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint

	// TODO(draychev): resolve the actual port number of this service
	port := endpoint.Port(constants.EnvoyInboundListenerPort)
	var computeKindObserver = map[computeKind]computeObserver{
		vm:   az.getVM,
		vmss: az.getVMSS,
	}

	for _, azID := range az.resolveService(svc) {
		glog.Infof("[%s] Getting Endpoints for service %s", packageName, svc)
		resourceGroup, kind, _, err := parseAzureID(azID)
		if err != nil {
			glog.Errorf("[%s] Unable to parse Azure URI %s: %s", packageName, azID, err)
			continue
		}

		if observer, ok := computeKindObserver[kind]; ok {
			var ips []net.IP
			var err error
			ips, err = observer(resourceGroup, azID)
			if err != nil {
				glog.Errorf("[%s] Could not fetch VMSS services: %s", packageName, err)
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

// ListServicesForServiceAccount retrieves the list of Services for the given service account
func (az Client) ListServicesForServiceAccount(svcAccount endpoint.NamespacedServiceAccount) []endpoint.NamespacedService {
	//TODO (snchh) : need to figure out the service account equivalnent for azure
	panic("NotImplemented")
}

func (az Client) run(stop <-chan struct{}) error {
	glog.V(level.Info).Infoln("Azure provider run started.")
	// TODO(draychev): implement pub/sub
	return nil
}

// GetAnnouncementsChannel returns the announcement channel for the Azure endponits provider.
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
	glog.V(level.Trace).Infof("[%s] Resolving service %s to an Azure URI", packageName, svc)
	var azureIDs []azureID
	service, exists, err := az.meshSpec.GetService(svc)
	if err != nil {
		glog.Errorf("[%s] Error fetching Kubernetes Endpoints from cache: %s", packageName, err)
		return azureIDs
	}
	if !exists {
		glog.Errorf("[%s] Error fetching Kubernetes Endpoints from cache: service %s does not exist", packageName, svc)
		return azureIDs
	}
	glog.V(level.Trace).Infof("[%s] Got the service: %+v", packageName, service)
	return matchServiceAzureResource(service, az.azureResourceClient.ListAzureResources())
}

type kv struct {
	k string
	v string
}

func matchServiceAzureResource(svc *corev1.Service, azureResourcesList []*osm.AzureResource) []azureID {
	glog.V(level.Trace).Infof("[%s] Match service %s to an AzureID", packageName, svc)
	azureResources := make(map[kv]*osm.AzureResource)
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
	glog.V(level.Trace).Infof("[%s] Found matches for service %s: %+v", packageName, svc, uris)
	return uris
}
