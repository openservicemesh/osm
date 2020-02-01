package lds

import (
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
)

func getFilterChain() []*listener.FilterChain {
	connManager, err := ptypes.MarshalAny(getConnManager())
	if err != nil {
		glog.Error("[LDS] Could not construct FilterChain: ", err)
		return nil
	}

	return []*listener.FilterChain{
		{
			Filters: []*listener.Filter{
				{
					Name: wellknown.HTTPConnectionManager,
					ConfigType: &listener.Filter_TypedConfig{
						TypedConfig: connManager,
					},
				},
			},
		},
	}
}
