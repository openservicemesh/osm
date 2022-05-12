package ads

import (
	"io"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
)

func receive(requests chan *xds_discovery.DiscoveryRequest, server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, proxy *envoy.Proxy, quit chan struct{}) {
	for {
		var request *xds_discovery.DiscoveryRequest
		request, recvErr := (*server).Recv()
		if recvErr != nil {
			defer close(requests)
			if status.Code(recvErr) == codes.Canceled || recvErr == io.EOF {
				log.Debug().Err(recvErr).Str("proxy", proxy.String()).Msg("gRPC Connection terminated")
				return
			}
			log.Error().Err(recvErr).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGRPCConnectionFailed)).
				Str("proxy", proxy.String()).Msg("gRPC Connection error")
			return
		}
		select {
		case <-(*server).Context().Done():
			log.Trace().Str("proxy", proxy.String()).Msgf("gRPC stream from proxy terminated")
			close(quit)
			return
		case requests <- request:
		}
		log.Debug().Str("proxy", proxy.String()).Msgf("Received DiscoveryRequest from proxy")
	}
}
