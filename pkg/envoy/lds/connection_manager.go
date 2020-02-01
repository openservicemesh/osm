package lds

import (
	accessLogV2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	connMgr "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
)

const (
	serverNameTODO  = "serverNameTODO"
	routeConfigName = "upstream"
	statPrefix      = "http"
)

func getConnManager() *connMgr.HttpConnectionManager {
	accessLog, err := ptypes.MarshalAny(getFileAccessLog())
	if err != nil {
		glog.Error("[LDS] Could con construct HttpConnectionManager struct: ", err)
		return nil
	}

	rds := getRDS()

	return &connMgr.HttpConnectionManager{
		ServerName: serverNameTODO,
		CodecType:  connMgr.HttpConnectionManager_AUTO,
		StatPrefix: statPrefix,
		RouteSpecifier: &connMgr.HttpConnectionManager_Rds{
			Rds: &connMgr.Rds{
				RouteConfigName: routeConfigName,
				ConfigSource:    rds,
			},
		},
		HttpFilters: []*connMgr.HttpFilter{
			{
				Name: wellknown.Router,
			},
		},
		AccessLog: []*accessLogV2.AccessLog{
			{
				Name: wellknown.FileAccessLog,
				ConfigType: &accessLogV2.AccessLog_TypedConfig{
					TypedConfig: accessLog,
				},
			},
		},
	}
}
