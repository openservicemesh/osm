package cds

import (
	"time"
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/utils"
	"github.com/deislabs/smc/pkg/logging"
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

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(server.Context())
	proxy := envoy.NewProxy(cn, ip)	
	s.catalog.RegisterProxy(envoy.NewProxy(cn, ip))
	glog.Infof("[%s][stream] Client connected: Subject CN=%s", serverName, cn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Periodic Updates -- useful for debugging
	go func() {
		counter := 0
		for {
			glog.V(log.LvlTrace).Infof("------------------------- %s Periodic Update %d -------------------------", serverName, counter)
			counter++
			s.announcements <- struct{}{}
			time.Sleep(5 * time.Second)
		}
	}()

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
			case <-s.announcements:
				// NOTE: This is deliberately only focused on providing MVP tools to run a TrafficRoute demo.
				glog.V(log.LvlInfo).Infof("[%s][stream] Received a change announcement! Updating all Envoy proxies.", serverName)
				resp, err := s.newDiscoveryResponse(proxy)
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