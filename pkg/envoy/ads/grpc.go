package ads

import (
	"io"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/openservicemesh/osm/pkg/envoy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func receive(requests chan xds_discovery.DiscoveryRequest, server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, proxy *envoy.Proxy, quit chan struct{}) {
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
			log.Trace().Msgf("[grpc] Received DiscoveryRequest from Envoy %s: %+v", proxy.GetCommonName(), request)
			requests <- *request
		} else {
			log.Warn().Msgf("[grpc] Unknown resource: %+v", request)
		}
	}
}
