package catalog

import (
	set "github.com/deckarep/golang-set"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/ingress"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/smi"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(namespaceController namespace.Controller, kubeClient kubernetes.Interface, meshSpec smi.MeshSpec, certManager certificate.Manager, ingressMonitor ingress.Monitor, broadcaster *Broadcaster, stop <-chan struct{}, cfg configurator.Configurator, endpointsProviders ...endpoint.Provider) *MeshCatalog {
	log.Info().Msg("Create a new Service MeshCatalog.")
	sc := MeshCatalog{
		endpointsProviders: endpointsProviders,
		meshSpec:           meshSpec,
		certManager:        certManager,
		ingressMonitor:     ingressMonitor,
		broadcaster:        broadcaster,
		configurator:       cfg,

		expectedProxies:      make(map[certificate.CommonName]expectedProxy),
		connectedProxies:     make(map[certificate.CommonName]connectedProxy),
		disconnectedProxies:  make(map[certificate.CommonName]disconnectedProxy),
		announcementChannels: set.NewSet(),

		// Kubernetes needed to determine what Services a pod that connects to XDS belongs to.
		// In multicluster scenarios this would be a map of cluster ID to Kubernetes client.
		// The certificate itself would contain the cluster ID making it easy to lookup the client in this map.
		kubeClient: kubeClient,

		namespaceController: namespaceController,
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

func (mc *MeshCatalog) getAnnouncementChannels() []announcementChannel {
	announcementChannels := []announcementChannel{
		{"MeshSpec", mc.meshSpec.GetAnnouncementsChannel()},
		{"CertManager", mc.certManager.GetAnnouncementsChannel()},
		{"IngressMonitor", mc.ingressMonitor.GetAnnouncementsChannel()},
		{"Ticker", mc.broadcaster.GetAnnouncementsChannel()},
		{"Namespace", mc.namespaceController.GetAnnouncementsChannel()},
	}
	for _, ep := range mc.endpointsProviders {
		annCh := announcementChannel{ep.GetID(), ep.GetAnnouncementsChannel()}
		announcementChannels = append(announcementChannels, annCh)
	}

	return announcementChannels
}
