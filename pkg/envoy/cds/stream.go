package cds

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

// StreamClusters handles streaming of the certs to the connected Envoy proxies
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

	
	// TODO(draychev): Use the Subject Common Name to identify the Envoy proxy and determine what service it belongs to.
	glog.Infof("[%s][stream] Client connected: Subject CN=%s", serverName, cn)

	if err := s.isConnectionAllowed(); err != nil {
		return err
	}

	// var recvErr error
	reqChannel := make(chan *xds.DiscoveryRequest)
	go receive(reqChannel, server)

	var announcements chan interface{}
	// Periodic Updates -- useful for debugging
	go func() {
		counter := 0
		for {
			glog.V(7).Infof("------------------------- %s Periodic Update %d -------------------------", serverName, counter)
			counter++
			announcements <- struct{}{}
			time.Sleep(5 * time.Second)
		}
	}()

	// TODO(draychev): filter on announcement type; only respond to Clusters change
	//announcements := s.catalog.GetAnnouncementChannel()

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
					glog.Errorf("[%s] Invalid Cluster Discovery request with no node", serverName)
					return errInvalidDiscoveryRequest
				}
	
				glog.Infof("[%s][incoming] Discovery Request from Envoy: %s", serverName, proxy.GetCommonName())
	
				response, err := s.newDiscoveryResponse(proxy)
				if err != nil {
					glog.Errorf("[%s] Failed constructing Cluster Discovery Response: %+v", serverName, err)
					return err
				}
				if err := server.Send(response); err != nil {
					glog.Errorf("[%s] Failed to send Cluster Discovery Response: %+v", serverName, err)
					return err
				}
				glog.Infof("[%s] Sent Clusters Discovery Response to client: %s", serverName, cn)
				glog.Infof("Deliberately sleeping for %d seconds...", sleepTime)
				time.Sleep(sleepTime * time.Second)
	
			case <-announcements:
				glog.Infof("[%s][outgoing] Clusters change announcement received.", serverName)
				response, err := s.newDiscoveryResponse(proxy)
				if err != nil {
					glog.Errorf("[%s] Failed constructing Cluster Discovery Response: %+v", serverName, err)
					return err
				}
				if err := server.Send(response); err != nil {
					glog.Infof("[%s] Failed to send Cluster Discovery Response: %+v", serverName, err)
					return err
				}
			}
		}

	}
	