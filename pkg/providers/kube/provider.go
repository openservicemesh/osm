package kube

import (
	"time"

	"github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/deislabs/smc/pkg/mesh"
	smcClient "github.com/deislabs/smc/pkg/smc_client/clientset/versioned"
)

func NewProvider(kubeClient *kubernetes.Clientset, smiClient *versioned.Clientset, azureResourceClient *smcClient.Clientset, namespaces []string, resyncPeriod time.Duration, announceChan *channels.RingChannel) mesh.ComputeProviderI {
	return newKubernetesClient(kubeClient, smiClient, azureResourceClient, namespaces, resyncPeriod, announceChan)
}

// GetIPs retrieves the list of IP addresses for the given service
func (kp Client) GetIPs(svc mesh.ServiceName) []mesh.IP {
	glog.Infof("[kubernetes] Getting IPs for service %s", svc)
	var ips []mesh.IP
	endpointsInterface, exist, err := kp.Caches.Endpoints.GetByKey(string(svc))
	if err != nil {
		glog.Error("Error fetching Kubernetes Endpoints from cache: ", err)
		return ips
	}

	if !exist {
		glog.Errorf("Error fetching Kubernetes Endpoints from cache: ServiceName %s does not exist", svc)
		return ips
	}

	if endpoints := endpointsInterface.(*v1.Endpoints); endpoints != nil {
		for _, endpoint := range endpoints.Subsets {
			for _, address := range endpoint.Addresses {
				ips = append(ips, mesh.IP(address.IP))
			}
		}
	}
	return ips
}

// Run executes informer collection.
func (kp *Client) Run(stopCh <-chan struct{}) error {
	glog.V(1).Infoln("k8s provider run started")
	var hasSynced []cache.InformerSynced

	if kp.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[friendlyName]cache.SharedInformer{
		"Endpoints":     kp.informers.Endpoints,
		"Services":      kp.informers.Services,
		"TrafficSplit":  kp.informers.TrafficSplit,
		"AzureResource": kp.informers.AzureResource,
	}

	var names []friendlyName
	for name, informer := range sharedInformers {
		names = append(names, name)
		glog.Infof("Starting informer: %s", name)
		go informer.Run(stopCh)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	glog.V(1).Infof("Waiting informers cache sync: %+v", names)
	if !cache.WaitForCacheSync(stopCh, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(kp.CacheSynced)

	glog.V(1).Infof("Cache sync finished for %+v", names)
	return nil
}
