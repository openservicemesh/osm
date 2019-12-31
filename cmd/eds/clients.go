package main

import (
	"time"

	smiClient "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/mesh"
	"github.com/deislabs/smc/pkg/mesh/providers"
	"github.com/deislabs/smc/pkg/providers/azure"
	"github.com/deislabs/smc/pkg/providers/kube"
)

func setupClients(announceChan *channels.RingChannel) (map[providers.Provider]mesh.ComputeProviderI, mesh.SpecI, mesh.ServiceCatalogI) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
	if err != nil {
		glog.Fatalf("Error gathering Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", *kubeConfigFile, err)
	}

	smiResources := smiClient.NewForConfigOrDie(kubeConfig)
	kubernetesProvider := kube.NewProvider(kubeConfig, smiResources, getNamespaces(), 1*time.Second, announceChan)
	azureProvider := azure.NewProvider(*subscriptionID, *resourceGroup, *namespace, *azureAuthFile, maxAuthRetryCount, retryPause, announceChan)
	stopChan := make(chan struct{})

	// Setup all Compute Providers -- these are Kubernetes, cloud provider virtual machines, etc.
	// TODO(draychev): How do we add multiple Kubernetes clusters? Multiple Azure subscriptions?
	computeProviders := map[providers.Provider]mesh.ComputeProviderI{
		providers.Kubernetes: kubernetesProvider,
		providers.Azure:      azureProvider,
	}

	// Run each provider -- starting the pub/sub system, which leverages the announceChan channel
	for providerType, provider := range computeProviders {
		if err := provider.Run(stopChan); err != nil {
			glog.Errorf("Could not start %s provider: %s", providerType, err)
			continue
		}
		if friendlyName, err := providers.GetFriendlyName(providerType); err == nil {
			glog.Infof("Started provider %s", friendlyName)
		} else {
			glog.Info("Started provider %d (could not find a friendly name for it)", providerType)
		}

	}

	// Mesh Spec Provider is something, which we query for SMI spec. Gives us the declaration of the service mesh.
	meshSpecProvider := kubernetesProvider

	// ServiceName Catalog is the facility, which we query to get the list of services, weights for traffic split etc.
	serviceCatalog := catalog.NewServiceCatalog(computeProviders, meshSpecProvider)

	return computeProviders, meshSpecProvider, serviceCatalog
}
