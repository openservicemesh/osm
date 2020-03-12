package lds

import (
	"context"
	"reflect"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/constants"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/route"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

// NewResponse creates a new Listener Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing listener Discovery Response for proxy: %s", packageName, proxy.GetCommonName())
	proxyServiceName := proxy.GetService()
	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeLDS),
	}

	clientConnManager, err := ptypes.MarshalAny(getHTTPConnectionManager(route.OutboundRouteConfig))
	if err != nil {
		glog.Errorf("[%s] Could not construct FilterChain: %s", packageName, err)
		return nil, err
	}

	outboundListenerName := "outbound_listener"
	clientListener := &xds.Listener{
		Name:    outboundListenerName,
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
	glog.Infof("Creating an %s for proxy %s for service %s: %+v", outboundListenerName, proxy.GetCommonName(), proxy.GetService(), clientListener)

	serverConnManager, err := ptypes.MarshalAny(getHTTPConnectionManager(route.InboundRouteConfig))
	if err != nil {
		glog.Errorf("[%s] Could not construct inbound listener FilterChain: %s", packageName, err)
		return nil, err
	}

	inboundListenerName := "inbound_listener"
	serverListener := &xds.Listener{
		Name:    inboundListenerName,
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
				/* --- This is commented out until we have a clear plan of how we are going to use FilterChainMatch ---
				// Source: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/listener/listener_components.proto
				// The FilterChainMatch uses SNI from mTLS to match against the provided list of ServerNames.
				// This ensures only clients authorized to talk to this listener are permitted to.
				FilterChainMatch: &listener.FilterChainMatch{
					ServerNames: []string{"smc/bookbuyer"}, // TODO(draychev): remove hard-coded demo value
				},
				*/
				TransportSocket: &envoy_api_v2_core.TransportSocket{
					Name: envoy.TransportSocketTLS,
					ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
						TypedConfig: envoy.GetDownstreamTLSContext(proxyServiceName),
					},
				},
			},
		},
	}
	glog.Infof("Created an %s for proxy %s for service %s: %+v", inboundListenerName, proxy.GetCommonName(), proxy.GetService(), serverListener)

	marshalledOutbound, err := ptypes.MarshalAny(clientListener)
	if err != nil {
		glog.Errorf("[%s] Failed to marshal outbound listener for proxy %s: %v", packageName, proxy.GetCommonName(), err)
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledOutbound)

	marshalledInbound, err := ptypes.MarshalAny(serverListener)
	if err != nil {
		glog.Errorf("[%s] Failed to marshal inbound listener for proxy %s: %v", packageName, proxy.GetCommonName(), err)
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledInbound)
	return resp, nil
}
