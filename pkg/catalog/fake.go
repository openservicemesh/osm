package catalog

import (
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/ingress"
	"github.com/open-service-mesh/osm/pkg/providers/kube"
	"github.com/open-service-mesh/osm/pkg/smi"
)

func NewFakeMeshCatalog() MeshCataloger {
	meshSpec := smi.NewFakeMeshSpecClient()
	certManager := tresor.NewFakeCertManager()
	ingressMonitor := ingress.NewFakeIngressMonitor()
	stop := make(<-chan struct{})
	endpointProviders := []endpoint.Provider{
		kube.NewFakeProvider(),
	}
	return NewMeshCatalog(meshSpec, certManager, ingressMonitor, stop, endpointProviders...)
}
