package ads

import (
	"io"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
)

func receive(requests chan xds_discovery.DiscoveryRequest, server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, proxy *envoy.Proxy, quit chan struct{}, proxyRegistry *registry.ProxyRegistry) {
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

		log.Trace().Msgf("[grpc] Received DiscoveryRequest from Envoy with certificate SerialNumber %s", proxy.GetCertificateSerialNumber())
		requests <- *request
	}
}
