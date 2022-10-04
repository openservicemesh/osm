package informers

import (
	"errors"
	"time"

	"k8s.io/client-go/tools/cache"
)

// InformerKey stores the different Informers we keep for K8s resources
type InformerKey string

const (
	// InformerKeyNamespace is the InformerKey for a Namespace informer
	InformerKeyNamespace InformerKey = "Namespace"
	// InformerKeyService is the InformerKey for a Service informer
	InformerKeyService InformerKey = "Service"
	// InformerKeyPod is the InformerKey for a Pod informer
	InformerKeyPod InformerKey = "Pod"
	// InformerKeyEndpoint is the InformerKey for a Endpoint informer
	InformerKeyEndpoint InformerKey = "Endpoint"
	// InformerKeyServiceAccount is the InformerKey for a ServiceAccount informer
	InformerKeyServiceAccount InformerKey = "ServiceAccount"
	// InformerKeySecret is the InformerKey for a Secret informer
	InformerKeySecret InformerKey = "Secret"

	// InformerKeyTrafficSplit is the InformerKey for a TrafficSplit informer
	InformerKeyTrafficSplit InformerKey = "TrafficSplit"
	// InformerKeyTrafficTarget is the InformerKey for a TrafficTarget informer
	InformerKeyTrafficTarget InformerKey = "TrafficTarget"
	// InformerKeyHTTPRouteGroup is the InformerKey for a HTTPRouteGroup informer
	InformerKeyHTTPRouteGroup InformerKey = "HTTPRouteGroup"
	// InformerKeyTCPRoute is the InformerKey for a TCPRoute informer
	InformerKeyTCPRoute InformerKey = "TCPRoute"

	// InformerKeyMeshConfig is the InformerKey for a MeshConfig informer
	InformerKeyMeshConfig InformerKey = "MeshConfig"
	// InformerKeyMeshRootCertificate is the InformerKey for a MeshRootCertificate informer
	InformerKeyMeshRootCertificate InformerKey = "MeshRootCertificate"
	// InformerKeyExtensionService is the InformerKey for an ExtensionService informer
	InformerKeyExtensionService InformerKey = "ExtensionService"

	// InformerKeyEgress is the InformerKey for a Egress informer
	InformerKeyEgress InformerKey = "Egress"
	// InformerKeyIngressBackend is the InformerKey for a IngressBackend informer
	InformerKeyIngressBackend InformerKey = "IngressBackend"
	// InformerKeyUpstreamTrafficSetting is the InformerKey for a UpstreamTrafficSetting informer
	InformerKeyUpstreamTrafficSetting InformerKey = "UpstreamTrafficSetting"
	// InformerKeyRetry is the InformerKey for a Retry informer
	InformerKeyRetry InformerKey = "Retry"
	// InformerKeyTelemetry is the InformerKey for a Telemetry informer
	InformerKeyTelemetry InformerKey = "Telemetry"
)

const (
	// DefaultKubeEventResyncInterval is the default resync interval for k8s events
	// This is set to 0 because we do not need resyncs from k8s client, and have our
	// own Ticker to turn on periodic resyncs.
	DefaultKubeEventResyncInterval = 0 * time.Second
)

var (
	errInitInformers = errors.New("informer not initialized")
	errSyncingCaches = errors.New("failed initial cache sync for informers")
)

// InformerCollection is an abstraction around a set of informers
// initialized with the clients stored in its fields. This data
// type should only be passed around as a pointer
type InformerCollection struct {
	informers map[InformerKey]cache.SharedIndexInformer
	meshName  string
}
