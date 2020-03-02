package lds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/constants"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/route"
	"github.com/deislabs/smc/pkg/smi"
)

// NewResponse creates a new Listener Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing listener Discovery Response for proxy: %s", serverName, proxy.GetCommonName())
	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeLDS),
	}

	clientConnManager, err := ptypes.MarshalAny(getHTTPConnectionManager(route.SourceRouteConfig))
	if err != nil {
		glog.Error("[LDS] Could not construct FilterChain: ", err)
		return nil, err
	}
	clientListener := &xds.Listener{
		Name:    "outbound_listener",
		Address: envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort),
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

	serverConnManager, err := ptypes.MarshalAny(getHTTPConnectionManager(route.DestinationRouteConfig))
	if err != nil {
		glog.Errorf("[%s] Could not construct inbound listener FilterChain: %s", serverName, err)
		return nil, err
	}

	serverListener := &xds.Listener{
		Name:    "inbound_listener",
		Address: envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort),
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
				TransportSocket: envoy.GetTransportSocketForServiceDownstream(string(proxy.GetService())),
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
	return resp, nil
}
