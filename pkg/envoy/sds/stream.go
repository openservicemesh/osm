package sds

import (
	"time"

	envoyv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/utils"
)

// StreamSecrets handles streaming of the certs to the connected Envoy proxies
func (s *Server) StreamSecrets(server v2.SecretDiscoveryService_StreamSecretsServer) error {
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

	// TODO(draychev): Use the Subject Common Name to identify the Envoy proxy and determine what service it belongs to.
	glog.Infof("[%s][stream] Client connected: Subject CN=%s", serverName, cn)

	if err := s.isConnectionAllowed(); err != nil {
		return err
	}

	/*
		// TODO(draychev): enable this once we have ServiceCatalog in s.
			// Register the newly connected Envoy proxy.
			connectedProxyIPAddress := net.IP("TBD")
			connectedProxyCertCommonName := certificate.CommonName("TBD")
			proxy := envoy.NewProxy(connectedProxyCertCommonName, connectedProxyIPAddress)
			s.catalog.RegisterProxy(proxy)
	*/

	reqChannel := make(chan *envoyv2.DiscoveryRequest)
	go receive(reqChannel, server)

	// TODO(draychev): filter on announcement type; only respond to secrets change
	announcements := s.catalog.GetAnnouncementChannel()

	for {
		select {
		case discoveryRequest, ok := <-reqChannel:
			if !ok {
				return errGrpcClosed
			}
			if discoveryRequest.ErrorDetail != nil {
				return errEnvoyError
			}
			if len(s.lastNonce) > 0 && discoveryRequest.ResponseNonce == s.lastNonce {
				continue
			}
			if discoveryRequest.Node == nil {
				glog.Errorf("[%s] Invalid Service Discovery request with no node", serverName)
				return errInvalidDiscoveryRequest
			}

			glog.Infof("[%s][incoming] Discovery Request from Envoy: %s", serverName, proxy.GetCommonName())

			response, err := s.newDiscoveryResponse(proxy)
			if err != nil {
				glog.Errorf("[%s] Failed constructing Secret Discovery Response: %+v", serverName, err)
				return err
			}
			if err := server.Send(response); err != nil {
				glog.Errorf("[%s] Failed to send Secret Discovery Response: %+v", serverName, err)
				return err
			}
			glog.Infof("[%s] Sent Secrets Discovery Response to client: %s", serverName, cn)
			glog.Infof("Deliberately sleeping for %d seconds...", sleepTime)
			time.Sleep(sleepTime * time.Second)

		case <-announcements:
			glog.Infof("[%s][outgoing] Secrets change announcement received.", serverName)
			response, err := s.newDiscoveryResponse(proxy)
			if err != nil {
				glog.Errorf("[%s] Failed constructing Secret Discovery Response: %+v", serverName, err)
				return err
			}
			if err := server.Send(response); err != nil {
				glog.Infof("[%s] Failed to send Secret Discovery Response: %+v", serverName, err)
				return err
			}
		}
	}

}
