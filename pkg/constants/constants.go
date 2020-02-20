package constants

const (
	// DefaultKubeNamespace is the default Kubernetes namespace.
	DefaultKubeNamespace = "default"

	// AzureProviderName is the string constant used for the ID of the Azure endpoints provider.
	// These strings identify the participating clusters / endpoint providers.
	// Ideally these should be not only the type of compute but also a unique identifier, like the FQDN of the cluster,
	// or the subscription within the cloud vendor.
	AzureProviderName = "Azure"

	// KubeProviderName is a string constant used for the ID string of the Kubernetes endpoints provider.
	KubeProviderName = "Kubernetes"

	// WildcardIPAddr is a string constant.
	WildcardIPAddr = "0.0.0.0"

	// EnvoyInboundListenerPort is the Envoy's inbound listener port number.
	EnvoyInboundListenerPort = 15003

	// EnvoyOutboundListenerPort is the Envoy's outbound listener port number.
	EnvoyOutboundListenerPort = 15001

	// HTTPPort is a port number constant.
	HTTPPort = 80
)
