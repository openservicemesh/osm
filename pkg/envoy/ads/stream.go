package ads

import (
	"context"
	"strconv"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/utils"
)

// StreamAggregatedResources handles streaming of the clusters to the connected Envoy proxies
func (s *Server) StreamAggregatedResources(server discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil, packageName)
	if err != nil {
		return errors.Wrap(err, "[%s] Could not start stream")
	}

	// TODO(draychev): check for envoy.ErrTooManyConnections

	glog.Infof("[%s] Client connected: Subject CN=%s", packageName, cn)

	// Register the newly connected proxy w/ the catalog.
	// TODO(draychev): this does not produce the correct IP address
	ip := utils.GetIPFromContext(server.Context())
	proxy := envoy.NewProxy(cn, ip)
	s.catalog.RegisterProxy(proxy)
	defer s.catalog.UnregisterProxy(proxy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requests := make(chan v2.DiscoveryRequest)
	go receive(requests, &server)

	for {
		select {
		case <-ctx.Done():
			return nil

		case discoveryRequest, ok := <-requests:
			if !ok {
				glog.Errorf("[%s] Proxy %s closed GRPC", packageName, proxy)
				return errGrpcClosed
			}

			if discoveryRequest.ErrorDetail != nil {
				glog.Errorf("[%s] Discovery request error from proxy %s: %s", packageName, proxy, discoveryRequest.ErrorDetail)
				return errEnvoyError
			}

			typeURL := envoy.TypeURI(discoveryRequest.TypeUrl)

			requestVersion, err := strconv.ParseUint(discoveryRequest.VersionInfo, 10, 64)

			glog.V(level.Debug).Infof("[%s] Incoming Discovery Request %s (nonce=%s; version=%d) from Envoy %s; last applied version: %d",
				packageName, discoveryRequest.TypeUrl, discoveryRequest.ResponseNonce, requestVersion, proxy.GetCommonName(), proxy.GetLastAppliedVersion(typeURL))
			glog.V(level.Debug).Infof("[%s] Last sent nonce=%s; last sent version=%d for Envoy %s",
				packageName, proxy.GetLastSentNonce(typeURL), proxy.GetLastSentVersion(typeURL), proxy.GetCommonName())

			proxy.SetLastAppliedVersion(typeURL, requestVersion)
			if err != nil && discoveryRequest.VersionInfo != "" {
				glog.Errorf("[%s] Error parsing %s discovery request VersionInfo (%s) from proxy %s: %s", packageName, typeURL, discoveryRequest.VersionInfo, proxy.GetCommonName(), err)
			}

			if requestVersion <= proxy.GetLastSentVersion(typeURL) {
				glog.V(level.Debug).Infof("[%s] %s Discovery Request VersionInfo (%d) <= last sent VersionInfo (%d); ACK", packageName, typeURL, requestVersion, proxy.GetLastSentVersion(typeURL))
				continue
			}

			lastNonce := proxy.GetLastSentNonce(typeURL)
			if lastNonce != "" && discoveryRequest.ResponseNonce == lastNonce {
				glog.V(level.Debug).Infof("[%s] Nothing changed since Nonce=%s", packageName, discoveryRequest.ResponseNonce)
				continue
			}

			if discoveryRequest.ResponseNonce != "" {
				glog.V(level.Debug).Infof("[%s] Received discovery request with Nonce=%s; matches=%t; proxy last Nonce=%s", packageName, discoveryRequest.ResponseNonce, discoveryRequest.ResponseNonce == lastNonce, lastNonce)
			}
			glog.Infof("[%s] Received discovery request <%s> from Envoy <%s> with Nonce=%s", packageName, discoveryRequest.TypeUrl, proxy, discoveryRequest.ResponseNonce)

			resp, err := s.newAggregatedDiscoveryResponse(proxy, &discoveryRequest)
			if err != nil {
				glog.Errorf("[%s] Error composing a DiscoveryResponse: %+v", packageName, err)
				continue
			}

			if err := server.Send(resp); err != nil {
				glog.Errorf("[%s] Error sending DiscoveryResponse: %+v", packageName, err)
			} else {
				glog.V(level.Debug).Infof("[%s] Sent Discovery Response %s to proxy %s: %s", packageName, resp.TypeUrl, proxy, resp)
			}

		case <-proxy.GetAnnouncementsChannel():
			glog.V(level.Info).Infof("[%s] Change detected - update all Envoys.", packageName)
			s.sendAllResponses(proxy, &server)
		}
	}
}
