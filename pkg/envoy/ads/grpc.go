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
				log.Debug().Err(recvErr).Msgf("[grpc] Connection terminated")
				return
			}
			log.Error().Err(recvErr).Msgf("[grpc] Connection error")
			return
		}
		if request.TypeUrl != "" {
			if !proxy.HasPodMetadata() {
				// Set the Pod metadata on the given proxy only once. This could arrive with the first few XDS requests.
				recordEnvoyPodMetadata(request, proxy, catalog)
			}
			log.Trace().Msgf("[grpc] Received DiscoveryRequest from Envoy with certificate SerialNumber %s", proxy.GetCertificateSerialNumber())
			requests <- *request
		} else {
			log.Warn().Msgf("[grpc] Received a request for an unknown TypeURL: %+v", request.TypeUrl)
		}
	}
}

func recordEnvoyPodMetadata(request *xds_discovery.DiscoveryRequest, proxy *envoy.Proxy, catalog catalog.MeshCataloger) {
	if request != nil && request.Node != nil {
		if meta, err := envoy.ParseEnvoyServiceNodeID(request.Node.Id); err != nil {
			log.Error().Err(err).Msgf("Error parsing Envoy Node ID: %s", request.Node.Id)
		} else {
			log.Trace().Msgf("Recorded metadata for Envoy with xDS Certificate SerialNumber=%s: podUID=%s, podNamespace=%s, serviceAccountName=%s, envoyNodeID=%s",
				proxy.GetCertificateSerialNumber(), meta.UID, meta.Namespace, meta.ServiceAccount, meta.EnvoyNodeID)

			// Set the Pod Metadata, which will be used in the RegisterProxy() invocation below!
			proxy.PodMetadata = meta

			// We call RegisterProxy again, for a second time, on the MeshCatalog to update the index on pod metadata
			catalog.RegisterProxy(proxy) // Second of Two invocations. First one was on establishing the gRPC stream.
		}
	}
}
