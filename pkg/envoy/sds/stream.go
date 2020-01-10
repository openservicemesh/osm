package sds

import (
	"io"

	envoyv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamSecrets handles streaming of the certs to the connected Envoy proxies
func (s *Server) StreamSecrets(stream v2.SecretDiscoveryService_StreamSecretsServer) error {
	glog.Info("[SDS] Starting SecretsStreamer...")
	var recvErr error
	var nodeID string

	if err := s.isConnectionAllowed(); err != nil {
		return err
	}

	reqChannel := make(chan *envoyv2.DiscoveryRequest, 1)

	go func() {
		defer close(reqChannel)
		for {
			var req *envoyv2.DiscoveryRequest

			req, recvErr = stream.Recv()
			if recvErr != nil {
				if status.Code(recvErr) == codes.Canceled || recvErr == io.EOF {
					glog.Infof("SDS: connection terminated %+v\n", recvErr)
					return
				}
				glog.Infof("SDS: connection terminated with errors %+v\n", recvErr)
				return
			}
			glog.Info("[SDS] Done!")
			reqChannel <- req
		}
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		glog.Info("Failed to create watcher:", err)
		return err
	}
	glog.Info("Created file system watcher...")
	defer watcher.Close()

	if err = watcher.Add(s.keysDirectory + certFileName); err != nil {
		glog.Errorf("Failed to add %s/%s to watcher: %+v\n", s.keysDirectory, certFileName, err)
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
				glog.Info("Invalid discovery request with no node")
				return errInvalidDiscoveryRequest
			}

			nodeID = discReq.Node.Id
			glog.Info("[SDS] Discovery Request from Envoy ID: ", nodeID)

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
				glog.Info("Failed to send:", err)
				return err
			}
		case ev := <-watcher.Events:
			glog.Infof("Got a file system watcher event...")
			if ev.Op == fsnotify.Remove || ev.Op == fsnotify.Rename {
				glog.Info("Key file is missing")
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
				glog.Info("Failed to send:", err)
				return err
			}
		case err := <-watcher.Errors:
			glog.Info("Watcher got error:", err)
			return err
		}
	}
}
