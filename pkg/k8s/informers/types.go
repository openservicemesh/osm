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
	// Kubernetes
	InformerKeyNamespace      InformerKey = "Namespace"
	InformerKeyService        InformerKey = "Service"
	InformerKeyPod            InformerKey = "Pod"
	InformerKeyEndpoints      InformerKey = "Endpoints"
	InformerKeyServiceAccount InformerKey = "ServiceAccount"

	// SMI
	InformerKeyTrafficSplit   InformerKey = "TrafficSplit"
	InformerKeyTrafficTarget  InformerKey = "TrafficTarget"
	InformerKeyHTTPRouteGroup InformerKey = "HTTPRouteGroup"
	InformerKeyTCPRoute       InformerKey = "TCPRoute"

	// Config
	InformerKeyMeshConfig          InformerKey = "MeshConfig"
	InformerKeyMeshRootCertificate InformerKey = "MeshRootCertificate"

	// Policy
	InformerKeyEgress                 InformerKey = "Egress"
	InformerKeyIngressBackend         InformerKey = "IngressBackend"
	InformerKeyUpstreamTrafficSetting InformerKey = "UpstreamTrafficSetting"
	InformerKeyRetry                  InformerKey = "Retry"
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
