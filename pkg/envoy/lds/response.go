package lds

import (
	"fmt"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log"
)

func (s *Server) newDiscoveryResponse(proxy envoy.Proxyer) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing listener Discovery Response for proxy: %s", serverName, proxy.GetCommonName())
	resp := &xds.DiscoveryResponse{
		TypeUrl: typeUrl,
	}

	listenerName := "the___listener"

	lisnr := &xds.Listener{
		Name:         listenerName,
		Address:      getAddress(),
		FilterChains: getFilterChain(),
	}
	marshalledListeners, err := ptypes.MarshalAny(lisnr)
	if err != nil {
		glog.Errorf("[%s] Failed to marshal listener for proxy %s: %v", serverName, proxy.GetCommonName(), err)
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledListeners)

	s.lastVersion = s.lastVersion + 1
	s.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = s.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", s.lastVersion)

	glog.V(log.LvlTrace).Infof("[%s] Constructed response: %+v", serverName, resp)

	return resp, nil
}
