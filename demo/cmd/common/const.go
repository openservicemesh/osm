package common

const (
	// Success is the string constant emitted at the end of the Bookbuyer logs when the test succeeded.
	Success = "SUCCESS"

	// Failure is the string constant emitted at the end of the Bookbuyer logs when the test failed.
	Failure = "FAILURE"

	// KubeConfigEnvVar is the environment variable holding path to kube config
	KubeConfigEnvVar = "KUBECONFIG"

	// KubeNamespaceEnvVar is the environment variable with the k8s namespace
	KubeNamespaceEnvVar = "K8S_NAMESPACE"

	// AppNamespacesEnvVar is the environment variable for a comma separated list of application namespaces
	AppNamespacesEnvVar = "APP_NAMESPACES"

	// OsmIDEnvVar is the environment variable for the namespace an OSM instance belongs to
	OsmIDEnvVar = "OSM_ID"

	// IsGithubEnvVar is the environment variable indicating whether this runs in Github CI.
	IsGithubEnvVar = "IS_GITHUB"

	// AggregatedDiscoveryServiceName is the name of the ADS service.
	AggregatedDiscoveryServiceName = "ads"

	// AggregatedDiscoveryServicePort is the port on which XDS listens for new connections.
	AggregatedDiscoveryServicePort = 15128

	// ContainerRegistryCredsEnvVar is the name of the environment variable storing the name of the container registry creds.
	ContainerRegistryCredsEnvVar = "CTR_REGISTRY_CREDS_NAME"

	// ContainerRegistryEnvVar is the name of the environment variable storing the container registry.
	ContainerRegistryEnvVar = "CTR_REGISTRY"

	// AzureSubscription is the name of the env var storing the azure subscription to watch.
	AzureSubscription = "AZURE_SUBSCRIPTION"

	// BooksBought header
	BooksBoughtHeader = "Booksbought"

	// Identity header
	IdentityHeader = "Identity"
)
