package ads

import (
	"io"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/envoy"
)

func receive(requests chan xds_discovery.DiscoveryRequest, server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, proxy *envoy.Proxy, quit chan struct{}, catalog catalog.MeshCataloger) {
	defer close(requests)
	defer close(quit)
	for {
		var request *xds_discovery.DiscoveryRequest
		request, recvErr := (*server).Recv()
		if recvErr != nil {
			if status.Code(recvErr) == codes.Canceled || recvErr == io.EOF {
				log.Error().Msgf("[grpc] Connection terminated: %+v", recvErr)
				return
			}
			log.Error().Msgf("[grpc] Connection terminated with error: %+v", recvErr)
			return
		}
		if request.TypeUrl != "" {
			if !proxy.HasPodMetadata() {
				// Set the Pod metadata on the given proxy only once. This could arrive with the first few XDS requests.
				recordEnvoyPodMetadata(request, proxy, catalog)
			}
			nodeID := ""
			if request.Node != nil {
				nodeID = request.Node.Id
			}
			log.Trace().Msgf("[grpc] Received DiscoveryRequest from Envoy with CN %s; Node ID: %s", proxy.GetCommonName(), nodeID)
			requests <- *request
		} else {
			log.Warn().Msgf("[grpc] Unknown resource: %+v", request)
		}
	}
}

func recordEnvoyPodMetadata(request *xds_discovery.DiscoveryRequest, proxy *envoy.Proxy, catalog catalog.MeshCataloger) {
	if request != nil && request.Node != nil {
		if meta, err := envoy.ParseEnvoyServiceNodeID(request.Node.Id); err != nil {
			log.Error().Err(err).Msgf("Error parsing Envoy Node ID: %s", request.Node.Id)
		} else {
			log.Trace().Msgf("Recorded metadata for Envoy %s: podUID=%s, podNamespace=%s, podIP=%s, serviceAccountName=%s, envoyNodeID=%s",
				proxy.CommonName, meta.UID, meta.Namespace, meta.IP, meta.ServiceAccount, meta.EnvoyNodeID)
			proxy.PodMetadata = meta

			// We call RegisterProxy again on the MeshCatalog to update the index on pod metadata
			catalog.RegisterProxy(proxy)
		}
	}
}
