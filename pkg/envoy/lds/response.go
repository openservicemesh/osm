package lds

import (
	"fmt"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	v23 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	v22 "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log"
)

func (s *Server) newDiscoveryResponse(proxy envoy.Proxyer) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing listener Discovery Response for proxy: %s", serverName, proxy.GetCommonName())
	resp := &xds.DiscoveryResponse{
		TypeUrl: typeUrl,
	}

	lbCluster := "servicelb" // the cluster the filter will redirect the traffic to.
	listenerAddress := "0.0.0.0"
	// TODO(draychev): figure this out from the service
	listenerPort := uint32(8080)

	accessLogConfigStruct := &v23.FileAccessLog{
		Path: "/dev/stdout",
	}

	filterStatPrefix := "ingress_tcp"

	accessLogConfig, err := util.MessageToStruct(accessLogConfigStruct)
	if err != nil {
		glog.Errorf("Failed to convert accessLog message %+v to struct", accessLogConfigStruct)
		panic(err)
	}

	filterConfigStruct := &v2.TcpProxy{
		StatPrefix: filterStatPrefix,
		ClusterSpecifier: &v2.TcpProxy_Cluster{
			Cluster: lbCluster,
		},
		AccessLog: []*v22.AccessLog{
			{
				Name: util.FileAccessLog,
				ConfigType: &v22.AccessLog_Config{
					Config: accessLogConfig,
				},
			},
		},
	}

	filterConfig, err := util.MessageToStruct(filterConfigStruct)
	if err != nil {
		glog.Errorf("Failed to convert filterConfig message %+v to struct", filterConfigStruct)
		panic(err)
	}

	lisnr := &xds.Listener{
		Name: "listener1",
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  listenerAddress,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: listenerPort,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name: util.TCPProxy,
						ConfigType: &listener.Filter_Config{
							Config: filterConfig,
						},
					},
				},
			},
		},
	}
	marshalledListeners, err := types.MarshalAny(lisnr)
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
