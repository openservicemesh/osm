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

	// EnvoyAdminPort is Envoy's admin port
	EnvoyAdminPort = 15000

	// EnvoyAdminPortName is Envoy's admin port name
	EnvoyAdminPortName = "proxy-admin"

	// EnvoyInboundListenerPort is Envoy's inbound listener port number.
	EnvoyInboundListenerPort = 15003

	// EnvoyInboundListenerPortName is Envoy's inbound listener port name.
	EnvoyInboundListenerPortName = "proxy-inbound"

	// EnvoyOutboundListenerPort is Envoy's outbound listener port number.
	EnvoyOutboundListenerPort = 15001

	// EnvoyOutboundListenerPortName is Envoy's outbound listener port name.
	EnvoyOutboundListenerPortName = "proxy-outbound"

	// EnvoyUID is the Envoy's User ID
	EnvoyUID int64 = 1337

	// CertCommonNameUUIDServiceDelimiter is the character used to delimit the UUID and service name in the certificate's CommonName
	CertCommonNameUUIDServiceDelimiter = ";"

	// NamespaceServiceDelimiter is the character used to delimit a namespace and a service name when used together
	NamespaceServiceDelimiter = "/"

	// InjectorWebhookPort is the port on which the sidecar injection webhook listens
	InjectorWebhookPort = 9090

	// RootCertPemStoreName is the name of the root certificate config store
	RootCertPemStoreName = "ca-rootcertpemstore"

	// RootCertPath is the path too the root certificate
	RootCertPath = "/etc/ssl/certs/root-cert.pem"

	// MetricsServerPort is the port on which OSM exposes its own metrics server
	MetricsServerPort = 9091

	// AggregatedDiscoveryServiceName is the name of the ADS service.
	AggregatedDiscoveryServiceName = "ads"

	// AggregatedDiscoveryServicePort is the port on which XDS listens for new connections.
	AggregatedDiscoveryServicePort = 15128
)
