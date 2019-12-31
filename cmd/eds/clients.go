package main

import (
	"time"

	smiClient "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/mesh"
	"github.com/deislabs/smc/pkg/providers/azure"
	"github.com/deislabs/smc/pkg/providers/kube"
	smcClient "github.com/deislabs/smc/pkg/smc_client/clientset/versioned"
)

// Categories of meshed service/compute providers
// TODO(draychev): further break down by k8s cluster, cloud subscription etc.
var (
	providerAzure      = "providerAzure"
	providerKubernetes = "providerKubernetes"
)

func setupClients(announceChan *channels.RingChannel) (map[string]mesh.ComputeProviderI, mesh.SpecI, mesh.ServiceCatalogI) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
	if err != nil {
		glog.Fatalf("Error gathering Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", *kubeConfigFile, err)
	}

	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	smiResourcesClient := smiClient.NewForConfigOrDie(kubeConfig)
	azureResourcesClient := smcClient.NewForConfigOrDie(kubeConfig)
	kubernetesProvider := kube.NewProvider(kubeClient, smiResourcesClient, azureResourcesClient, getNamespaces(), 1*time.Second, announceChan)
	azureProvider := azure.NewProvider(*subscriptionID, *resourceGroup, *namespace, *azureAuthFile, maxAuthRetryCount, retryPause, announceChan)
	stopChan := make(chan struct{})

	// Setup all Compute Providers -- these are Kubernetes, cloud provider virtual machines, etc.
	// TODO(draychev): How do we add multiple Kubernetes clusters? Multiple Azure subscriptions?
	computeProviders := map[string]mesh.ComputeProviderI{
		providerKubernetes: kubernetesProvider,
		providerAzure:      azureProvider,
	}

	// Run each provider -- starting the pub/sub system, which leverages the announceChan channel
	for providerType, provider := range computeProviders {
		if err := provider.Run(stopChan); err != nil {
			glog.Errorf("Could not start %s provider: %s", providerType, err)
			continue
		}
		glog.Infof("Started provider %s", providerType)
	}

	// Mesh Spec Provider is something, which we query for SMI spec. Gives us the declaration of the service mesh.
	meshSpecProvider := kubernetesProvider

	// ServiceName Catalog is the facility, which we query to get the list of services, weights for traffic split etc.
	serviceCatalog := catalog.NewServiceCatalog(computeProviders, meshSpecProvider)

	return computeProviders, meshSpecProvider, serviceCatalog
}
