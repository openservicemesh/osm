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

	// EnvoyInboundPrometheusListenerPortName is Envoy's inbound listener port name for prometheus.
	EnvoyInboundPrometheusListenerPortName = "proxy-metrics"

	// EnvoyOutboundListenerPort is Envoy's outbound listener port number.
	EnvoyOutboundListenerPort = 15001

	// EnvoyOutboundListenerPortName is Envoy's outbound listener port name.
	EnvoyOutboundListenerPortName = "proxy-outbound"

	// EnvoyUID is the Envoy's User ID
	EnvoyUID int64 = 1337

	// LocalhostIPAddress is the local host address.
	LocalhostIPAddress = "127.0.0.1"

	// EnvoyMetricsCluster is the cluster name of the Prometheus metrics cluster
	EnvoyMetricsCluster = "envoy-metrics-cluster"

	// EnvoyZipkinCluster is the name of the Zipkin cluster.
	EnvoyZipkinCluster = "envoy-zipkin-cluster"

	// EnvoyZipkinEndpoint zipkin endpoint.
	EnvoyZipkinEndpoint = "/api/v2/spans"

	// EnvoyZipkinAddress is the address of the Zipkin server.
	EnvoyZipkinAddress = "zipkin"

	// EnvoyZipkinPort is the Zipkin port number.
	// TODO(draychev): Move this to the Config CRD
	EnvoyZipkinPort = 9411

	// EnvoyPrometheusInboundListenerPort is Envoy's inbound listener port number for prometheus
	EnvoyPrometheusInboundListenerPort = 15010

	// InjectorWebhookPort is the port on which the sidecar injection webhook listens
	InjectorWebhookPort = 9090

	// MetricsServerPort is the port on which OSM exposes its own metrics server
	MetricsServerPort = 9091

	// OSMControllerName is the name of the OSM Controller (formerly ADS service).
	OSMControllerName = "osm-controller"

	// OSMControllerPort is the port on which XDS listens for new connections.
	OSMControllerPort = 15128

	// PrometheusScrapePath is the path for prometheus to scrap envoy metrics from
	PrometheusScrapePath = "/stats/prometheus"

	// PrometheusVirtualHostName is used to setup a remote endpoint for comms
	PrometheusVirtualHostName = "prometheus_envoy_admin"

	// PrometheusDefaultName when not given, is a default prometheus name given by the platform itself
	PrometheusDefaultName = "osm-prometheus"

	// CertificationAuthorityCommonName is the CN used for the root certificate for OSM.
	CertificationAuthorityCommonName = "Open Service Mesh Certification Authority"

	// CertificationAuthorityRootValidityPeriod is when the root certificate expires
	CertificationAuthorityRootValidityPeriod = 87600 * time.Hour // a decade

	// XDSCertificateValidityPeriod is the TTL of the certificates used for Envoy to xDS communication.
	XDSCertificateValidityPeriod = 87600 * time.Hour // a decade

	// RegexMatchAll is a regex pattern match for all
	RegexMatchAll = ".*"

	// WildcardHTTPMethod is a wildcard for all HTTP methods
	WildcardHTTPMethod = "*"

	// OSMKubeResourceMonitorAnnotation is the key of the annotation used to monitor a K8s resource
	OSMKubeResourceMonitorAnnotation = "openservicemesh.io/monitored-by"

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

	// ----- Environment Variables

	// EnvVarLogKubernetesEvents is the name of the env var instructing the event handlers whether to log at all (true/false)
	EnvVarLogKubernetesEvents = "OSM_LOG_KUBERNETES_EVENTS"

	// EnvVarHumanReadableLogMessages is an environment variable, which when set to "true" enables colorful human-readable log messages.
	EnvVarHumanReadableLogMessages = "OSM_HUMAN_DEBUG_LOG"

	// ClusterWeightAcceptAll is the weight for a cluster that accepts 100 percent of traffic sent to it
	ClusterWeightAcceptAll = 100

	// PrometheusDefaultRetentionTime is the default days for which data is retained in prometheus
	PrometheusDefaultRetentionTime = "15d"

	// DomainDelimiter is a delimiter used in representing domains
	DomainDelimiter = "."
)
