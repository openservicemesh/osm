package catalog

import (
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/endpoint/providers/kube"
	"github.com/open-service-mesh/osm/pkg/ingress"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/namespace"
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
	
	namespaceCtrlr := namespace.NewFakeNamespaceController([]string{osmNamespace})

	return NewMeshCatalog(namespaceCtrlr, kubeClient, meshSpec, certManager, ingressMonitor, stop, cfg, endpointProviders...)
}
