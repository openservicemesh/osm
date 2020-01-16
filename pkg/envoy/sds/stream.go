package sds

import (
	"io"

	"github.com/deislabs/smc/pkg/utils"
	envoyv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	serverName = "SDS"
)

// StreamSecrets handles streaming of the certs to the connected Envoy proxies
func (s *Server) StreamSecrets(stream v2.SecretDiscoveryService_StreamSecretsServer) error {
	glog.Infof("[%s] Starting SecretsStreamer", serverName)

	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(stream.Context(), nil, serverName)
	if err != nil {
		return errors.Wrap(err, "[%s] Could not start stream")
	}

	// TODO(draychev): Use the Subject Common Name to identify the Envoy proxy and determine what service it belongs to.
	glog.Infof("[%s][stream] Client connected: Subject CN=%+v", serverName, cn)

	var recvErr error
	var nodeID string

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

	reqChannel := make(chan *envoyv2.DiscoveryRequest, 1)

	go func() {
		defer close(reqChannel)
		for {
			var req *envoyv2.DiscoveryRequest

			req, recvErr = stream.Recv()
			if recvErr != nil {
				if status.Code(recvErr) == codes.Canceled || recvErr == io.EOF {
					glog.Infof("[%s] connection terminated %+v", serverName, recvErr)
					return
				}
				glog.Infof("[%s] connection terminated with errors %+v", serverName, recvErr)
				return
			}
			glog.Infof("[%s] Done!", serverName)
			reqChannel <- req
		}
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		glog.Infof("[%s] Failed to create watcher: %+v", serverName, err)
		return err
	}
	glog.Infof("[%s] Created file system watcher", serverName)
	defer watcher.Close()

	if err = watcher.Add(s.keysDirectory + certFileName); err != nil {
		glog.Errorf("[%s] Failed to add %s/%s to watcher: %+v", serverName, s.keysDirectory, certFileName, err)
		return err
	}

	for {
		select {
		case discReq, ok := <-reqChannel:
			if !ok {
				return recvErr
			}
			if discReq.ErrorDetail != nil {
				return errEnvoyError
			}
			if len(s.lastNonce) > 0 && discReq.ResponseNonce == s.lastNonce {
				continue
			}
			if discReq.Node == nil {
				glog.Infof("[%s] Invalid discovery request with no node", serverName)
				return errInvalidDiscoveryRequest
			}

			nodeID = discReq.Node.Id
			glog.Infof("[%s] Discovery Request from Envoy ID: %s", serverName, nodeID)

			secret, err := getSecretItem(s.keysDirectory)
			if err != nil {
				return err
			}
			response, err := s.sdsDiscoveryResponse(secret, nodeID)
			if err != nil {
				glog.Info(err)
				return err
			}
			if err := stream.Send(response); err != nil {
				glog.Infof("[%s] Failed to send: %+v", serverName, err)
				return err
			}
		case ev := <-watcher.Events:
			glog.Infof("[%s] Got a file system watcher event...", serverName)
			if ev.Op == fsnotify.Remove || ev.Op == fsnotify.Rename {
				glog.Infof("[%s] Key file is missing", serverName)
				return errKeyFileMissing
			}
			secret, err := getSecretItem(s.keysDirectory)
			if err != nil {
				return err
			}
			response, err := s.sdsDiscoveryResponse(secret, nodeID)
			if err != nil {
				glog.Info(err)
				return err
			}
			if err := stream.Send(response); err != nil {
				glog.Infof("[%s] Failed to send: %+v", serverName, err)
				return err
			}
		case err := <-watcher.Errors:
			glog.Infof("[%s] Watcher got error: %+v", serverName, err)
			return err
		}
	}
}
