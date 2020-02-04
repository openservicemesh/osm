package cds

import (
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

func getHttp2() *core.Http2ProtocolOptions {
	return &core.Http2ProtocolOptions{}
}
