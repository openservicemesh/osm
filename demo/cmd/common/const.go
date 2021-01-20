package common

const (
	// Success is the string constant emitted at the end of the Bookbuyer/Bookthief logs when the test succeeded.
	Success = "MAESTRO! THIS TEST SUCCEEDED!"

	// Failure is the string constant emitted at the end of the Bookbuyer/Bookthief logs when the test failed.
	Failure = "MAESTRO, WE HAVE A PROBLEM! THIS TEST FAILED!"

	// KubeNamespaceEnvVar is the environment variable with the k8s namespace
	KubeNamespaceEnvVar = "K8S_NAMESPACE"

	// ContainerRegistryCredsEnvVar is the name of the environment variable storing the name of the container registry creds.
	ContainerRegistryCredsEnvVar = "CTR_REGISTRY_CREDS_NAME"

	// ContainerRegistryEnvVar is the name of the environment variable storing the container registry.
	ContainerRegistryEnvVar = "CTR_REGISTRY"

	// ContainerTag is the name of the environment variable storing the container tag for the images to be used.
	ContainerTag = "CTR_TAG"

	// BooksBoughtHeader is the header returned by the bookstore and observed by the bookbuyer.
	BooksBoughtHeader = "Booksbought"

	// IdentityHeader is the header returned by the bookstore and observed by the bookbuyer.
	IdentityHeader = "Identity"

	// PrometheusRetention is the environment variable for retention time of prometheus data
	PrometheusRetention = "PROMETHEUS_RETENTION_TIME"

	// BookstoreNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookstoreNamespaceEnvVar = "BOOKSTORE_NAMESPACE"

	// BookwarehouseNamespaceEnvVar is the environment variable for the Bookwarehouse namespace.
	BookwarehouseNamespaceEnvVar = "BOOKWAREHOUSE_NAMESPACE"

	// BookthiefExpectedResponseCodeEnvVar is the environment variable for Bookthief's expected HTTP response code
	BookthiefExpectedResponseCodeEnvVar = "BOOKTHIEF_EXPECTED_RESPONSE_CODE"

	// EnableEgressEnvVar is the environment variable to enable egress requests in the demo
	EnableEgressEnvVar = "ENABLE_EGRESS"
)
