package constants

import "time"

const (
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

	// EnvoyInboundPromethuesListenerPortName is Envoy's inbound listener port name for prometheus.
	EnvoyInboundPromethuesListenerPortName = "proxy-metrics"

	// EnvoyOutboundListenerPort is Envoy's outbound listener port number.
	EnvoyOutboundListenerPort = 15001

	// EnvoyOutboundListenerPortName is Envoy's outbound listener port name.
	EnvoyOutboundListenerPortName = "proxy-outbound"

	// EnvoyUID is the Envoy's User ID
	EnvoyUID int64 = 1337

	// LocalhostIPAddress is the local host address.
	LocalhostIPAddress = "127.0.0.1"

	// EnvoyAdminCluster is the cluster name for Envoy's Admin
	EnvoyAdminCluster = "envoy-admin-cluster"

	// EnvoyPrometheusInboundListenerPort is Envoy's inbound listener port number for prometheus
	EnvoyPrometheusInboundListenerPort = 15010

	// InjectorWebhookPort is the port on which the sidecar injection webhook listens
	InjectorWebhookPort = 9090

	// MetricsServerPort is the port on which OSM exposes its own metrics server
	MetricsServerPort = 9091

	// AggregatedDiscoveryServiceName is the name of the ADS service.
	AggregatedDiscoveryServiceName = "ads"

	// AggregatedDiscoveryServicePort is the port on which XDS listens for new connections.
	AggregatedDiscoveryServicePort = 15128

	// PrometheusScrapePath is the path for prometheus to scrap envoy metrics from
	PrometheusScrapePath = "/stats/prometheus"

	// CertificationAuthorityCommonName is the CN used for the root certificate for OSM.
	CertificationAuthorityCommonName = "Open Service Mesh Certification Authority"

	// CertificationAuthorityRootExpiration is when the root certificate expires
	CertificationAuthorityRootExpiration = 87600 * time.Hour // a decade

	// RegexMatchAll is a regex pattern match for all
	RegexMatchAll = ".*"

	// WildcardHTTPMethod is a wildcard for all HTTP methods
	WildcardHTTPMethod = "*"

	// OSMKubeResourceMonitorAnnotation is the key of the annotation used to monitor a K8s resource
	OSMKubeResourceMonitorAnnotation = "openservicemesh.io/monitor"

	// KubernetesOpaqueSecretCAKey is the key which holds the CA bundle in a Kubernetes secret.
	KubernetesOpaqueSecretCAKey = "ca.crt"

	// KubernetesOpaqueSecretRootPrivateKeyKey is the key which holds the CA's private key in a Kubernetes secret.
	KubernetesOpaqueSecretRootPrivateKeyKey = "private.key"

	// KubernetesOpaqueSecretCAExpiration is the key which holds the CA's expiration in a Kubernetes secret.
	KubernetesOpaqueSecretCAExpiration = "expiration"

	// EnvoyUniqueIDLabelName is the label applied to pods with the unique ID of the Envoy sidecar.
	EnvoyUniqueIDLabelName = "osm-envoy-uid"

	// TimeDateLayout is the layout for time.Parse used in this repo
	TimeDateLayout = "2006-01-02T15:04:05.000Z"
)
