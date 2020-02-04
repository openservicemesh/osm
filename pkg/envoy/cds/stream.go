package cds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	sleepTime = 5
)

// StreamClusters handles streaming of the clusters to the connected Envoy proxies
func (s *Server) StreamClusters(server xds.ClusterDiscoveryService_StreamClustersServer) error {
	glog.Infof("[%s] Starting StreamClusters", serverName)

	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil, serverName)
	if err != nil {
		return errors.Wrap(err, "[%s] Could not start stream")
	}

	glog.Infof("[%s][stream] Client connected: Subject CN=%s", serverName, cn)

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(server.Context())
	proxy := envoy.NewProxy(cn, ip)
	msgCh := s.catalog.ProxyRegister(proxy.GetID())
	defer s.catalog.ProxyUnregister(proxy.GetID())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		request, err := server.Recv()
		if err != nil {
			return errors.Wrap(err, "recv")
		}

		if request.TypeUrl != typeUrl {
			glog.Errorf("[%s][stream] Unknown TypeUrl: %s", serverName, request.TypeUrl)
			return errUnknownTypeURL
		}

	Run:
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-msgCh:
				// NOTE: This is deliberately only focused on providing MVP tools to run a TrafficRoute demo.
				glog.V(log.LvlInfo).Infof("[%s][stream] Received a change msg! Updating all Envoy proxies.", serverName)
				resp, err := s.newClusterDiscoveryResponse(proxy)
				if err != nil {
					glog.Errorf("[%s][stream] Failed composing a DiscoveryResponse: %+v", serverName, err)
					return err
				}
				if err := server.Send(resp); err != nil {
					glog.Errorf("[%s][stream] Error sending DiscoveryResponse: %+v", serverName, err)
				}
				break Run
			}
		}
	}
}
