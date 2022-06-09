package informers

import (
	"errors"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	smiTrafficAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"

	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
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
	// InformerKeyEndpoints is the InformerKey for a Endpoints informer
	InformerKeyEndpoints InformerKey = "Endpoints"
	// InformerKeyServiceAccount is the InformerKey for a ServiceAccount informer
	InformerKeyServiceAccount InformerKey = "ServiceAccount"

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

	// InformerKeyEgress is the InformerKey for a Egress informer
	InformerKeyEgress InformerKey = "Egress"
	// InformerKeyIngressBackend is the InformerKey for a IngressBackend informer
	InformerKeyIngressBackend InformerKey = "IngressBackend"
	// InformerKeyUpstreamTrafficSetting is the InformerKey for a UpstreamTrafficSetting informer
	InformerKeyUpstreamTrafficSetting InformerKey = "UpstreamTrafficSetting"
	// InformerKeyRetry is the InformerKey for a Retry informer
	InformerKeyRetry InformerKey = "Retry"
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

type informer struct {
	customStore cache.Store
	informer    cache.SharedIndexInformer
}

type InformerCollection struct {
	informers             map[InformerKey]*informer
	meshName              string
	kubeClient            kubernetes.Interface
	smiTrafficSplitClient smiTrafficSplitClient.Interface
	smiTrafficSpecClient  smiTrafficSpecClient.Interface
	smiAccessClient       smiTrafficAccessClient.Interface
	configClient          configClientset.Interface
	policyClient          policyClientset.Interface
	selectedInformers     []InformerKey
	customStores          map[InformerKey]cache.Store
}

type informerInit func()

func (i *informer) GetStore() cache.Store {
	if i.customStore != nil {
		return i.customStore
	}

	return i.informer.GetStore()
}

func (i *informer) HasSynced() bool {
	return i.informer.HasSynced()
}

func (i *informer) Run(stop <-chan struct{}) {
	i.informer.Run(stop)
}
