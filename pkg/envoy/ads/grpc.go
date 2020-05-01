package ads

import (
	"io"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func receive(requests chan v2.DiscoveryRequest, server *xds.AggregatedDiscoveryService_StreamAggregatedResourcesServer) {
	defer close(requests)
	for {
		var request *v2.DiscoveryRequest
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
			requests <- *request
		} else {
			log.Warn().Msgf("[grpc] Unknown resource: %s", request.String())
		}
	}
}
