package lds

import (
	"fmt"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/constants"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
)

const (
	serverName = "LDS"
)

func (s *Server) NewListenerDiscoveryResponse(proxy envoy.Proxyer) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing listener Discovery Response for proxy: %s", serverName, proxy.GetCommonName())
	resp := &xds.DiscoveryResponse{
		TypeUrl: envoy.TypeLDS,
	}

	clientConnManager, err := ptypes.MarshalAny(getRdsHTTPClientConnectionFilter())
	if err != nil {
		glog.Error("[LDS] Could not construct FilterChain: ", err)
		return nil, err
	}
	clientListener := &xds.Listener{
		Name:    "outbound_listener",
		Address: envoy.GetAddress(constants.WildcardIpAddr, constants.EnvoyOutboundListenerPort),
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name: wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: clientConnManager,
						},
					},
				},
			},
		},
	}

	serverConnManager, err := ptypes.MarshalAny(getRdsHTTPServerConnectionFilter())
	if err != nil {
		glog.Error("[LDS] Could not construct inbound listener FilterChain: ", err)
		return nil, err
	}

	serverListener := &xds.Listener{
		Name:    "inbound_listener",
		Address: envoy.GetAddress(constants.WildcardIpAddr, constants.EnvoyInboundListenerPort),
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name: wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: serverConnManager,
						},
					},
				},
				// TODO(draychev): enable "tls_context.require_client_certificate: true"
				TransportSocket: envoy.GetTransportSocketForServiceDownstream("server_cert"), // TODO(draychev): remove hard-coded cert name
			},
		},
	}

	marshalledOutbound, err := ptypes.MarshalAny(clientListener)
	if err != nil {
		glog.Errorf("[%s] Failed to marshal outbound listener for proxy %s: %v", serverName, proxy.GetCommonName(), err)
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledOutbound)

	marshalledInbound, err := ptypes.MarshalAny(serverListener)
	if err != nil {
		glog.Errorf("[%s] Failed to marshal inbound listener for proxy %s: %v", serverName, proxy.GetCommonName(), err)
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledInbound)

	s.lastVersion = s.lastVersion + 1
	s.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = s.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", s.lastVersion)
	glog.V(level.Trace).Infof("[%s] Constructed response: %+v", serverName, resp)
	return resp, nil
}
