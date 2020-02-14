package constants

const (
	DefaultKubeNamespace = "default"

	// These strings identify the participating clusters / endpoint providers.
	// Ideally these should be not only the type of compute but also a unique identifier, like the FQDN of the cluster,
	// or the subscription within the cloud vendor.
	AzureProviderName = "Azure"
	KubeProviderName  = "Kubernetes"

	WildcardIpAddr = "0.0.0.0"

	// Listener ports configured on Envoy for handling inbound and outbound traffic
	EnvoyInboundListenerPort   = 15003
	EnvoyOutboundListenerPort = 15001

	HttpPort = 80
)
