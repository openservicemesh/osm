package catalog

import (
	"fmt"
	"net/http"
	"os"
	"time"

	mapset "github.com/deckarep/golang-set"
	"k8s.io/client-go/kubernetes"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/ingress"
	"github.com/open-service-mesh/osm/pkg/smi"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(kubeClient kubernetes.Interface, meshSpec smi.MeshSpec, certManager certificate.Manager, ingressMonitor ingress.Monitor, stop <-chan struct{}, endpointsProviders ...endpoint.Provider) *MeshCatalog {
	log.Info().Msg("Create a new Service MeshCatalog.")
	sc := MeshCatalog{
		endpointsProviders: endpointsProviders,
		meshSpec:           meshSpec,
		certManager:        certManager,
		ingressMonitor:     ingressMonitor,

		servicesCache:        make(map[endpoint.WeightedService][]endpoint.Endpoint),
		certificateCache:     make(map[endpoint.NamespacedService]certificate.Certificater),
		expectedProxies:      make(map[certificate.CommonName]expectedProxy),
		connectedProxies:     make(map[certificate.CommonName]connectedProxy),
		announcementChannels: mapset.NewSet(),
		serviceAccountsCache: make(map[endpoint.NamespacedServiceAccount][]endpoint.NamespacedService),

		// Kubernetes needed to determine what Services a pod that connects to XDS belongs to.
		// In multicluster scenarios this would be a map of cluster ID to Kubernetes client.
		// The certificate itself would contain the cluster ID making it easy to lookup the client in this map.
		kubeClient: kubeClient,
	}

	for _, announcementChannel := range sc.getAnnouncementChannels() {
		sc.announcementChannels.Add(announcementChannel)

	}

	sc.refreshCache()

	go sc.repeater()
	return &sc
}

// GetDebugInfo returns an HTTP handler for OSM debug endpoint.
func (mc *MeshCatalog) GetDebugInfo() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(draychev): convert to CLI flag
		if value, ok := os.LookupEnv("OSM_ENABLE_DEBUG"); ok && value == "true" {
			_, _ = fmt.Fprintf(w, "hello\n")
		}
	})
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
