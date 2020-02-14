package ads

import (
	"context"
	"fmt"
	"github.com/deislabs/smc/pkg/log/level"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	maxConnections = 10
)

// StreamAggregates handles streaming of the clusters to the connected Envoy proxies
func (s *Server) StreamAggregatedResources(server discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	glog.Infof("[%s] Starting StreamAggregates", serverName)

	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil, serverName)
	if err != nil {
		return errors.Wrap(err, "[%s] Could not start stream")
	}

	// TODO(draychev): check for envoy.ErrTooManyConnections

	glog.Infof("[%s][stream] Client connected: Subject CN=%s", serverName, cn)

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(server.Context())
	proxy := s.catalog.RegisterProxy(cn, ip)
	defer s.catalog.UnregisterProxy(proxy.GetID())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requests := make(chan *v2.DiscoveryRequest)
	go receive(requests, server)

	s.sendAllResponses(proxy, server)

	for {
		select {
		case <-ctx.Done():
			return nil

		case discoveryRequest, ok := <-requests:
			glog.Infof("[%s][incoming] Discovery Request %s from Envoy %s", serverName, discoveryRequest.TypeUrl, proxy.GetCommonName())
			if !ok {
				glog.Errorf("[%s] Proxy %s closed GRPC", serverName, proxy.GetCommonName())
				return errGrpcClosed
			}
			if discoveryRequest.ErrorDetail != nil {
				glog.Errorf("[%s] Discovery request error from proxy %s: %s", serverName, proxy.GetCommonName(), discoveryRequest.ErrorDetail)
				return errEnvoyError
			}
			if len(s.lastNonce) > 0 && discoveryRequest.ResponseNonce == s.lastNonce {
				glog.Warningf("[%s] Nothing changed", serverName)
				continue
			}
			resp, err := s.newAggregatedDiscoveryResponse(proxy, discoveryRequest)
			if err != nil {
				glog.Errorf("[%s][stream] Failed composing a DiscoveryResponse: %+v", serverName, err)
				return err
			}
			if err := server.Send(resp); err != nil {
				glog.Errorf("[%s][stream] Error sending DiscoveryResponse: %+v", serverName, err)
			}

		case <-proxy.GetAnnouncementsChannel():
			glog.V(level.Info).Infof("[%s][stream] Change detected - update all Envoys.", serverName)
			s.sendAllResponses(proxy, server)
		}
	}
}

func (s *Server) sendAllResponses(proxy envoy.Proxyer, server discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) {
	// Order is important
	uris := []string{envoy.TypeCDS, envoy.TypeEDS, envoy.TypeLDS, envoy.TypeRDS, envoy.TypeSDS}
	for _, uri := range uris {
		request := &v2.DiscoveryRequest{TypeUrl: uri}
		if discoveryResponse, err := s.newAggregatedDiscoveryResponse(proxy, request); err != nil {
			glog.Error(err)
			continue
		} else {
			if err := server.Send(discoveryResponse); err != nil {
				glog.Errorf("[%s][stream] Error sending DiscoveryResponse %s: %+v", serverName, uri, err)
			}
		}
	}
}

func (s *Server) newAggregatedDiscoveryResponse(proxy envoy.Proxyer, request *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	var err error
	var response *v2.DiscoveryResponse

	glog.V(level.Info).Infof("[%s][stream] Received discovery request: %s", serverName, request.TypeUrl)

	switch request.TypeUrl {
	case envoy.TypeEDS:
		weightedServices, err := s.catalog.ListEndpoints("TBD")
		if err != nil {
			glog.Errorf("[%s][stream] Failed listing endpoints: %+v", serverName, err)
			return nil, err
		}
		glog.Infof("[%s][stream] WeightedServices: %+v", serverName, weightedServices)
		response, err = s.edsServer.NewEndpointDiscoveryResponse(weightedServices)
	case envoy.TypeCDS:
		response, err = s.cdsServer.NewClusterDiscoveryResponse(proxy)
	case envoy.TypeRDS:
		glog.V(level.Info).Infof("[%s][stream] Received a change message! Updating all Envoy proxies.", serverName)
		trafficPolicies, err := s.catalog.ListTrafficRoutes("TBD")
		if err != nil {
			glog.Errorf("[%s][stream] Failed listing routes: %+v", serverName, err)
			return nil, err
		}
		glog.Infof("[%s][stream] trafficPolicies: %+v", serverName, trafficPolicies)
		response, err = s.rdsServer.NewRouteDiscoveryResponse(trafficPolicies)
	case envoy.TypeLDS:
		response, err = s.ldsServer.NewListenerDiscoveryResponse(proxy)
	case envoy.TypeSDS:
		response, err = s.sdsServer.NewSecretDiscoveryResponse(proxy)
	default:
		glog.Errorf("Responder for TypeUrl %s is not implemented", request.TypeUrl)
		return nil, errUnknownTypeURL
	}

	if err != nil {
		glog.Errorf("Error creating %s response: %s", request.TypeUrl, err)
		return nil, errCreatingResponse
	}

	s.lastVersion = s.lastVersion + 1
	s.lastNonce = string(time.Now().Nanosecond())
	response.Nonce = s.lastNonce
	response.VersionInfo = fmt.Sprintf("v%d", s.lastVersion)
	glog.V(level.Trace).Infof("[%s] Constructed response: %+v", serverName, response)
	return response, nil
}
