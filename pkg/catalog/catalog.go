package catalog

import (
	"time"

	set "github.com/deckarep/golang-set"
	"k8s.io/client-go/kubernetes"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/ingress"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/namespace"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(namespaceCtrlr namespace.Controller, kubeClient kubernetes.Interface, meshSpec smi.MeshSpec, certManager certificate.Manager, ingressMonitor ingress.Monitor, stop <-chan struct{}, cfg configurator.Configurator, endpointsProviders ...endpoint.Provider) *MeshCatalog {
	log.Info().Msg("Create a new Service MeshCatalog.")
	sc := MeshCatalog{
		endpointsProviders: endpointsProviders,
		meshSpec:           meshSpec,
		certManager:        certManager,
		ingressMonitor:     ingressMonitor,
		configurator:       cfg,

		expectedProxies:      make(map[certificate.CommonName]expectedProxy),
		connectedProxies:     make(map[certificate.CommonName]connectedProxy),
		disconnectedProxies:  make(map[certificate.CommonName]disconnectedProxy),
		announcementChannels: set.NewSet(),

		// Kubernetes needed to determine what Services a pod that connects to XDS belongs to.
		// In multicluster scenarios this would be a map of cluster ID to Kubernetes client.
		// The certificate itself would contain the cluster ID making it easy to lookup the client in this map.
		kubeClient: kubeClient,

		namespaceCtrlr: 	namespaceCtrlr,
	}

	for _, announcementChannel := range sc.getAnnouncementChannels() {
		sc.announcementChannels.Add(announcementChannel)

	}

	go sc.repeater()
	return &sc
}

// GetSMISpec returns a MeshCatalog's SMI Spec
func (mc *MeshCatalog) GetSMISpec() smi.MeshSpec {
	return mc.meshSpec
}

// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
func (mc *MeshCatalog) RegisterNewEndpoint(smi.ClientIdentity) {
	// TODO(draychev): implement
	panic("NotImplemented")
}

func (mc *MeshCatalog) getAnnouncementChannels() []announcementChannel {
	ticking := make(chan interface{})
	announcementChannels := []announcementChannel{
		{"MeshSpec", mc.meshSpec.GetAnnouncementsChannel()},
		{"CertManager", mc.certManager.GetAnnouncementsChannel()},
		{"IngressMonitor", mc.ingressMonitor.GetAnnouncementsChannel()},
		{"Ticker", ticking},
		{"Namespace", mc.namespaceCtrlr.GetAnnouncementsChannel()},
	}
	for _, ep := range mc.endpointsProviders {
		annCh := announcementChannel{ep.GetID(), ep.GetAnnouncementsChannel()}
		announcementChannels = append(announcementChannels, annCh)
	}

	go func() {
		ticker := time.NewTicker(updateAtLeastEvery)
		ticking <- ticker.C
	}()
	return announcementChannels
}
