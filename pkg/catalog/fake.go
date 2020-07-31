package catalog

import (
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/endpoint/providers/kube"
	"github.com/openservicemesh/osm/pkg/ingress"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/namespace"
)

// NewFakeMeshCatalog creates a new struct implementing catalog.MeshCataloger interface used for testing.
func NewFakeMeshCatalog(kubeClient kubernetes.Interface) *MeshCatalog {
	meshSpec := smi.NewFakeMeshSpecClient()
	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	ingressMonitor := ingress.NewFakeIngressMonitor()
	stop := make(<-chan struct{})
	endpointProviders := []endpoint.Provider{
		kube.NewFakeProvider(),
	}

	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"
	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
	
	namespaceController := namespace.NewFakeNamespaceController([]string{osmNamespace})

	return NewMeshCatalog(namespaceController, kubeClient, meshSpec, certManager, ingressMonitor, stop, cfg, endpointProviders...)
}
