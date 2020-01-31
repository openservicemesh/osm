package cds

import (
	"io"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v22 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func receive(reqChannel chan *v2.DiscoveryRequest, server v22.SecretDiscoveryService_StreamSecretsServer) {
	defer close(reqChannel)
	for {
		var request *v2.DiscoveryRequest
		request, recvErr := server.Recv()
		if recvErr != nil {
			if status.Code(recvErr) == codes.Canceled || recvErr == io.EOF {
				glog.Errorf("[%s][grpc] Connection terminated: %+v", serverName, recvErr)
				return
			}
			glog.Errorf("[%s][grpc] Connection terminated with error: %+v", serverName, recvErr)
			return
		}
		glog.Infof("[%s][grpc] Done!", serverName)
		reqChannel <- request
	}
}
