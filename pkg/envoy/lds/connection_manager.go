package lds

import (
	accesslogconfig "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	accessLogV2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	connMgr "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	structpb "github.com/golang/protobuf/ptypes/struct"
)

const (
	serverNameTODO  = "serverNameTODO"
	routeConfigName = "upstream"
	statPrefix      = "http"
	accessLogPath = "/dev/stdout"
)

func getConnManager() *connMgr.HttpConnectionManager {
	accessLog, err := ptypes.MarshalAny(getFileAccessLog())
	if err != nil {
		glog.Error("[LDS] Could con construct HttpConnectionManager struct: ", err)
		return nil
	}

	rdsSource := getRDSSource()

	return &connMgr.HttpConnectionManager{
		ServerName: serverNameTODO,
		CodecType:  connMgr.HttpConnectionManager_AUTO,
		StatPrefix: statPrefix,
		RouteSpecifier: &connMgr.HttpConnectionManager_Rds{
			Rds: &connMgr.Rds{
				RouteConfigName: routeConfigName,
				ConfigSource:    rdsSource,
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

func getFileAccessLog() * accesslogconfig.FileAccessLog {
	accessLogger := &accesslogconfig.FileAccessLog{
		Path: accessLogPath,
		AccessLogFormat: &accesslogconfig.FileAccessLog_JsonFormat{
			JsonFormat: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"start_time":            pbStringValue(`%START_TIME%`),
					"method":                pbStringValue(`%REQ(:METHOD)%`),
					"path":                  pbStringValue(`%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%`),
					"protocol":              pbStringValue(`%PROTOCOL%`),
					"response_code":         pbStringValue(`%RESPONSE_CODE%`),
					"response_code_details": pbStringValue(`%RESPONSE_CODE_DETAILS%`),
					"time_to_first_byte":    pbStringValue(`%RESPONSE_DURATION%`),
					"upstream_cluster":      pbStringValue(`%UPSTREAM_CLUSTER%`),
					"response_flags":        pbStringValue(`%RESPONSE_FLAGS%`),
					"bytes_received":        pbStringValue(`%BYTES_RECEIVED%`),
					"bytes_sent":            pbStringValue(`%BYTES_SENT%`),
					"duration":              pbStringValue(`%DURATION%`),
					"upstream_service_time": pbStringValue(`%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%`),
					"x_forwarded_for":       pbStringValue(`%REQ(X-FORWARDED-FOR)%`),
					"user_agent":            pbStringValue(`%REQ(USER-AGENT)%`),
					"request_id":            pbStringValue(`%REQ(X-REQUEST-ID)%`),
					"authority":             pbStringValue(`%REQ(:AUTHORITY)%`),
					"upstream_host":         pbStringValue(`%UPSTREAM_HOST%`),
				},
			},
		},
	}
	return accessLogger
}

func pbStringValue(v string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StringValue{
			StringValue: v,
		},
	}
}