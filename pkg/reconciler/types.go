package reconciler

import (
	"fmt"

	clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("reconciler")

const (
	// CrdInformerKey lookup identifier
	CrdInformerKey k8s.InformerKey = "CRDInformerKey"

	// MutatingWebhookInformerKey lookup identifier
	MutatingWebhookInformerKey k8s.InformerKey = "MutatingWebhookConfigInformerKey"

	// ValidatingWebhookInformerKey lookup identifier
	ValidatingWebhookInformerKey k8s.InformerKey = "ValidatingWebhookConfigInformerKey"

	// nameIndex is the lookup name for the most comment index function, which is to index by the name field
	nameIndex string = "name"
)

// informerCollection is the type holding the collection of informers we keep
type informerCollection map[k8s.InformerKey]cache.SharedIndexInformer

// client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster
type client struct {
	meshName        string
	osmVersion      string
	kubeClient      kubernetes.Interface
	apiServerClient clientset.Interface
	informers       informerCollection
}

// metaNameIndexFunc is a default index function that indexes based on an object's name
func metaNameIndexFunc(obj interface{}) ([]string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %w", err)
	}
	return []string{meta.GetName()}, nil
}
