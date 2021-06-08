package catalog

import (
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/ingress"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/provider"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/ticker"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(kubeController k8s.Controller, kubeClient kubernetes.Interface, meshSpec smi.MeshSpec, certManager certificate.Manager, ingressMonitor ingress.Monitor, policyController policy.Controller, stop <-chan struct{}, cfg configurator.Configurator, Providers ...provider.Provider) *MeshCatalog {
	log.Info().Msg("Create a new Service MeshCatalog.")
	mc := MeshCatalog{
		Providers:        Providers,
		meshSpec:         meshSpec,
		certManager:      certManager,
		ingressMonitor:   ingressMonitor,
		policyController: policyController,
		configurator:     cfg,

		// Kubernetes needed to determine what Services a pod that connects to XDS belongs to.
		// In multicluster scenarios this would be a map of cluster ID to Kubernetes client.
		// The certificate itself would contain the cluster ID making it easy to lookup the client in this map.
		kubeClient:     kubeClient,
		kubeController: kubeController,
	}

	go mc.dispatcher()
	ticker.InitTicker(cfg)

	return &mc
}

// GetKubecontroller returns the kubecontroller instance handling the current cluster
func (mc *MeshCatalog) GetKubecontroller() k8s.Controller {
	return mc.kubeController
}
