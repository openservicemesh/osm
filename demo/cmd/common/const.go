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

	// OsmIDEnvVar is the environment variable for the namespace an OSM instance belongs to
	OsmIDEnvVar = "OSM_ID"

	// IsGithubEnvVar is the environment variable indicating whether this runs in Github CI.
	IsGithubEnvVar = "IS_GITHUB"

	// ContainerRegistryCredsEnvVar is the name of the environment variable storing the name of the container registry creds.
	ContainerRegistryCredsEnvVar = "CTR_REGISTRY_CREDS_NAME"

	// ContainerRegistryEnvVar is the name of the environment variable storing the container registry.
	ContainerRegistryEnvVar = "CTR_REGISTRY"

	// ContainerTag is the name of the environment variable storing the container tag for the images to be used.
	ContainerTag = "CTR_TAG"

	// AzureSubscription is the name of the env var storing the azure subscription to watch.
	AzureSubscription = "AZURE_SUBSCRIPTION"

	// BooksBoughtHeader is the header returned by the bookstore and observed by the bookbuyer.
	BooksBoughtHeader = "Booksbought"

	// IdentityHeader is the header returned by the bookstore and observed by the bookbuyer.
	IdentityHeader = "Identity"

	// PrometheusVar is the environment variable for promethues service name
	PrometheusVar = "PROMETHEUS_SVC"

	// HumanReadableLogMessagesEnvVar is an environment variable, which when set to "true" enables colorful human-readable log messages.
	HumanReadableLogMessagesEnvVar = "OSM_HUMAN_DEBUG_LOG"

	// BookstoreNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookstoreNamespaceEnvVar = "BOOKSTORE_NAMESPACE"
)
