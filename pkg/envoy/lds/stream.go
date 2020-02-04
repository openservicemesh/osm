package lds

import (
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	sleepTime = 5
)

// StreamListeners handles streaming of the listeners to the connected Envoy proxies
func (s *Server) StreamListeners(server xds.ListenerDiscoveryService_StreamListenersServer) error {
	glog.Infof("[%s] Starting StreamListeners", serverName)

	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil, serverName)
	if err != nil {
		return errors.Wrapf(err, "[%s] Could not start StreamListeners", serverName)
	}

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(server.Context())
	proxy := envoy.NewProxy(cn, ip)
	s.catalog.RegisterProxy(envoy.NewProxy(cn, ip))

	// TODO(draychev): Use the Subject Common Name to identify the Envoy proxy and determine what service it belongs to.
	glog.Infof("[%s][stream] Client connected: Subject CN=%s", serverName, cn)

	if err := s.isConnectionAllowed(); err != nil {
		return err
	}

	// var recvErr error
	reqChannel := make(chan *xds.DiscoveryRequest)
	go receive(reqChannel, server)

	// TODO(draychev): filter on announcement type; only respond to Listeners change
	announcements := s.catalog.GetAnnouncementChannel()

	for {
		select {
		case discoveryRequest, ok := <-reqChannel:
			if !ok {
				return errGrpcClosed
			}
			if discoveryRequest.ErrorDetail != nil {
				return errDiscoveryRequest
			}
			if len(s.lastNonce) > 0 && discoveryRequest.ResponseNonce == s.lastNonce {
				continue
			}
			if discoveryRequest.Node == nil {
				glog.Errorf("[%s] Invalid Listener Discovery request with no node", serverName)
				return errInvalidDiscoveryRequest
			}

			glog.Infof("[%s][incoming] Discovery Request from Envoy: %s", serverName, proxy.GetCommonName())

			response, err := s.newDiscoveryResponse(proxy)
			if err != nil {
				glog.Errorf("[%s] Failed constructing Listener Discovery Response: %+v", serverName, err)
				return err
			}
			if err := server.Send(response); err != nil {
				glog.Errorf("[%s] Failed to send Listener Discovery Response: %+v", serverName, err)
				return err
			}
			glog.Infof("[%s] Sent Listeners Discovery Response to client: %s", serverName, cn)
			glog.Infof("Deliberately sleeping for %d seconds...", sleepTime)
			time.Sleep(sleepTime * time.Second)

		case <-announcements:
			glog.Infof("[%s][outgoing] Listeners change announcement received.", serverName)
			response, err := s.newDiscoveryResponse(proxy)
			if err != nil {
				glog.Errorf("[%s] Failed constructing Listener Discovery Response: %+v", serverName, err)
				return err
			}
			if err := server.Send(response); err != nil {
				glog.Infof("[%s] Failed to send Listener Discovery Response: %+v", serverName, err)
				return err
			}
		}
	}

}
