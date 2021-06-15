package catalog

import (
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/ingress"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/ticker"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(kubeController k8s.Controller, meshSpec smi.MeshSpec, certManager certificate.Manager, ingressMonitor ingress.Monitor, policyController policy.Controller, stop <-chan struct{}, cfg configurator.Configurator, serviceProviders []service.Provider, endpointsProviders []endpoint.Provider) *MeshCatalog {
	log.Info().Msg("Create a new Service MeshCatalog.")
	mc := MeshCatalog{
		serviceProviders:   serviceProviders,
		endpointsProviders: endpointsProviders,
		meshSpec:           meshSpec,
		certManager:        certManager,
		ingressMonitor:     ingressMonitor,
		policyController:   policyController,
		configurator:       cfg,

		kubeController: kubeController,
	}

	go mc.dispatcher()
	ticker.InitTicker(cfg)

	return &mc
}

// GetKubeController returns the kube controller instance handling the current cluster
func (mc *MeshCatalog) GetKubeController() k8s.Controller {
	return mc.kubeController
}
