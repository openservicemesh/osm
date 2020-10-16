package azure

import (
	"fmt"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"

	osm "github.com/openservicemesh/osm/pkg/apis/azureresource/v1"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
)

// ListEndpointsForService implements endpoints.Provider interface and returns the IP addresses and Ports for the given mesh service.
func (az Client) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint

	// TODO(draychev): resolve the actual port number of this service
	port := endpoint.Port(constants.EnvoyInboundListenerPort)
	var computeKindObserver = map[computeKind]computeObserver{
		vm:   az.getVM,
		vmss: az.getVMSS,
	}

	for _, azID := range az.resolveService(svc) {
		log.Info().Msgf("Getting Endpoints for service %s", svc)
		resourceGroup, kind, _, err := parseAzureID(azID)
		if err != nil {
			log.Error().Err(err).Msgf("Unable to parse Azure URI %s", azID)
			continue
		}

		if observer, ok := computeKindObserver[kind]; ok {
			var ips []net.IP
			var err error
			ips, err = observer(resourceGroup, azID)
			if err != nil {
				log.Error().Err(err).Msg("Could not fetch VMSS services")
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

// GetServicesForServiceAccount retrieves a list of services for the given service account.
func (az Client) GetServicesForServiceAccount(svcAccount service.K8sServiceAccount) ([]service.MeshService, error) {
	//TODO (snchh) : need to figure out the service account equivalnent for azure
	panic("NotImplemented")
}

func (az Client) run(stop <-chan struct{}) error {
	log.Info().Msg("Azure provider run started.")
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
	azureIDPathLen := 9 // See above
	chunks := strings.Split(string(id), "/")
	if len(chunks) != azureIDPathLen {
		return "", "", "", errIncorrectAzureURI
	}
	resGroup := resourceGroup(chunks[4])
	kind := computeKind(fmt.Sprintf("%s/%s", chunks[6], chunks[7]))
	name := computeName(chunks[8])
	return resGroup, kind, name, nil
}

func (az *Client) resolveService(svc service.MeshService) []azureID {
	log.Trace().Msgf("Resolving service %s to an Azure URI", svc)
	var azureIDs []azureID
	k8sService := az.kubeController.GetService(svc)
	if k8sService == nil {
		log.Error().Msgf("Error fetching Kubernetes Service for MeshService %s", svc)
		return azureIDs
	}
	log.Trace().Msgf("Got the service: %+v", k8sService)
	return matchServiceAzureResource(k8sService, az.azureResourceClient.ListAzureResources())
}

type kv struct {
	k string
	v string
}

func matchServiceAzureResource(svc *corev1.Service, azureResourcesList []*osm.AzureResource) []azureID {
	log.Trace().Msgf("Match service %s to an AzureID", svc)
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
	log.Trace().Msgf("Found matches for service %s: %+v", svc, uris)
	return uris
}
