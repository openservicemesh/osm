package kube

import (
	"fmt"
	"time"

	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	smiClient "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	smc "github.com/deislabs/smc/pkg/apis/azureresource/v1"
	"github.com/deislabs/smc/pkg/mesh"
	smcClient "github.com/deislabs/smc/pkg/smc_client/clientset/versioned"
)

// We have a few different k8s clients. This identifies these in logs.
const kubernetesClientName = "MeshSpec"

// NewMeshSpecClient creates the Kubernetes client, which retrieves SMI specific CRDs.
func NewMeshSpecClient(kubeConfig *rest.Config, namespaces []string, resyncPeriod time.Duration, announceChan *channels.RingChannel, stopChan chan struct{}) mesh.SpecI {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	smiClientset := smiClient.NewForConfigOrDie(kubeConfig)
	azureResourceClient := smcClient.NewForConfigOrDie(kubeConfig)
	k8sClient := NewClient(kubeClient, smiClientset, azureResourceClient, namespaces, resyncPeriod, announceChan, kubernetesClientName)
	err := k8sClient.Run(stopChan)
	if err != nil {
		glog.Fatalf("Could not start %s client: %s", kubernetesClientName, err)
	}
	return k8sClient
}

// GetTrafficSplitWeight retrieves the weight for the given service
func (c *Client) GetTrafficSplitWeight(target mesh.ServiceName, delegate mesh.ServiceName) (int, error) {
	item, exists, err := c.caches.TrafficSplit.Get(target)
	if err != nil {
		glog.Errorf("[%s] Error retrieving %v from TrafficSplit cache", kubernetesClientName, target)
		return 0, errRetrievingFromCache
	}
	if !exists {
		glog.Errorf("[%s] %v does not exist in TrafficSplit cache", kubernetesClientName, target)
		return 0, errNotInCache
	}
	ts := item.(v1alpha2.TrafficSplit)
	for _, be := range ts.Spec.Backends {
		if be.Service == string(delegate) {
			return be.Weight, nil
		}
	}
	glog.Errorf("[MeshSpec] Was looking for delegate %s for target service %s but did not find it", delegate, target)
	return 0, errBackendNotFound
}

// ListTrafficSplits returns the list of traffic splits.
func (c *Client) ListTrafficSplits() []*v1alpha2.TrafficSplit {
	var trafficSplits []*v1alpha2.TrafficSplit
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*v1alpha2.TrafficSplit)
		trafficSplits = append(trafficSplits, split)
	}
	return trafficSplits
}

// ListServices lists the services observed from the given compute provider
func (c *Client) ListServices() []mesh.ServiceName {
	// TODO(draychev): split the namespace and the service kubernetesClientName -- for non-kubernetes services we won't have namespace
	var services []mesh.ServiceName
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*v1alpha2.TrafficSplit)
		namespacedServiceName := fmt.Sprintf("%s/%s", split.Namespace, split.Spec.Service)
		services = append(services, mesh.ServiceName(namespacedServiceName))
		for _, backend := range split.Spec.Backends {
			namespacedServiceName := fmt.Sprintf("%s/%s", split.Namespace, backend.Service)
			services = append(services, mesh.ServiceName(namespacedServiceName))
		}
	}
	return services
}

// GetComputeIDForService returns the collection of compute platforms / clusters, which form the given Mesh Service.
func (c *Client) GetComputeIDForService(svc mesh.ServiceName) []mesh.ComputeID {
	var clusters []mesh.ComputeID
	serviceInterface, exist, err := c.caches.Services.GetByKey(string(svc))
	if err != nil {
		glog.Error("Error fetching Kubernetes Endpoints from cache: ", err)
		return clusters
	}

	if !exist {
		glog.Errorf("Error fetching Kubernetes Endpoints from cache: ServiceName %s does not exist", svc)
		return clusters
	}

	if c.caches.AzureResource == nil {
		//TODO(draychev): Should this be a Fatal?
		glog.Error("AzureResource Kubernetes Cache is incorrectly setup")
		return clusters
	}

	var azureResourcesList []*smc.AzureResource
	for _, azureResourceInterface := range c.caches.AzureResource.List() {
		azureResourcesList = append(azureResourcesList, azureResourceInterface.(*smc.AzureResource))
	}

	clusters = append(clusters, matchServiceAzureResource(serviceInterface.(*v1.Service), azureResourcesList))

	// TODO(draychev): populate list w/ !AzureResource

	return clusters
}

type kv struct {
	k string
	v string
}

func matchServiceAzureResource(svc *v1.Service, azureResourcesList []*smc.AzureResource) mesh.ComputeID {
	azureResources := make(map[kv]*smc.AzureResource)
	for _, azRes := range azureResourcesList {
		for k, v := range azRes.ObjectMeta.Labels {
			azureResources[kv{k, v}] = azRes
		}
	}
	computeID := mesh.ComputeID{}
	if service := svc; service != nil {
		for k, v := range service.ObjectMeta.Labels {
			if azRes, ok := azureResources[kv{k, v}]; ok && azRes != nil {
				computeID.AzureID = mesh.AzureID(azRes.Spec.ResourceID)
			}
		}
	}
	return computeID
}
