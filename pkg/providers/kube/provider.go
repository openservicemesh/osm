package kube

import (
	"time"

	"github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/deislabs/smc/pkg/mesh"
	smcClient "github.com/deislabs/smc/pkg/smc_client/clientset/versioned"
)

// NewProvider creates a new Kubernetes cluster/compute provider, which will inform SMC of Endpoints for a given service.
func NewProvider(kubeConfig *rest.Config, namespaces []string, resyncPeriod time.Duration, announceChan *channels.RingChannel, providerIdent string) mesh.ComputeProviderI {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	// smiClient and azureResourceClient are used for SMI spec observation only
	// these are not needed for the ComputeProviderI use-case
	var smiClient *versioned.Clientset
	var azureResourceClient *smcClient.Clientset
	return NewClient(kubeClient, smiClient, azureResourceClient, namespaces, resyncPeriod, announceChan, providerIdent)
}

// GetIPs retrieves the list of IP addresses for the given service
func (c Client) GetIPs(svc mesh.ServiceName) []mesh.IP {
	glog.Infof("[%s] Getting IPs for service %s on Kubernetes", c.providerIdent, svc)
	var ips []mesh.IP
	endpointsInterface, exist, err := c.caches.Endpoints.GetByKey(string(svc))
	if err != nil {
		glog.Errorf("[%s] Error fetching Kubernetes Endpoints from cache: %s", c.providerIdent, err)
		return ips
	}

	if !exist {
		glog.Errorf("[%s] Error fetching Kubernetes Endpoints from cache: ServiceName %s does not exist", c.providerIdent, svc)
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
func (c *Client) Run(stopCh <-chan struct{}) error {
	glog.V(1).Infoln("Kubernetes Compute Provider started")
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[friendlyName]cache.SharedInformer{
		"Endpoints":     c.informers.Endpoints,
		"Services":      c.informers.Services,
		"TrafficSplit":  c.informers.TrafficSplit,
		"AzureResource": c.informers.AzureResource,
	}

	var names []friendlyName
	for name, informer := range sharedInformers {
		// Depending on the use-case, some Informers from the collection may not have been initialized.
		if informer == nil {
			continue
		}
		names = append(names, name)
		glog.Info("Starting informer: ", name)
		go informer.Run(stopCh)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	glog.V(1).Infof("Waiting informers cache sync: %+v", names)
	if !cache.WaitForCacheSync(stopCh, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	glog.V(1).Infof("Cache sync finished for %+v", names)
	return nil
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interface: ComputeProviderI
func (c *Client) GetID() string {
	return c.providerIdent
}
