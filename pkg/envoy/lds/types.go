// Package lds implements Envoy's Listener Discovery Service (LDS).
package lds

import (
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/service"

	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

var (
	log = logger.New("envoy/lds")
)

type listenerBuilder struct {
	name                      string
	proxyIdentity             identity.ServiceIdentity
	address                   *xds_core.Address
	trafficDirection          xds_core.TrafficDirection
	trustDomain               string
	permissiveMesh            bool
	permissiveEgress          bool
	outboundMeshTrafficPolicy *trafficpolicy.OutboundMeshTrafficPolicy
	inboundMeshTrafficPolicy  *trafficpolicy.InboundMeshTrafficPolicy
	egressTrafficPolicy       *trafficpolicy.EgressTrafficPolicy
	ingressTrafficPolicies    []*trafficpolicy.IngressTrafficPolicy
	trafficTargets            []trafficpolicy.TrafficTargetWithRoutes
	wasmStatsHeaders          map[string]string
	httpTracingEndpoint       string
	extAuthzConfig            *auth.ExtAuthConfig
	activeHealthCheck         bool
	sidecarSpec               configv1alpha2.SidecarSpec
	filBuilder                *filterBuilder

	listenerFilters []*xds_listener.ListenerFilter
}

type httpConnManagerBuilder struct {
	statsPrefix         string
	routeConfigName     string
	filters             []*xds_hcm.HttpFilter
	tracing             *xds_hcm.HttpConnectionManager_Tracing
	localReplyConfig    *xds_hcm.LocalReplyConfig
	routerFilter        *xds_hcm.HttpFilter
	httpGlobalRateLimit *policyv1alpha1.HTTPGlobalRateLimitSpec
}

type tcpProxyBuilder struct {
	statsPrefix      string
	cluster          string
	weightedClusters []service.WeightedCluster
}

type filterBuilder struct {
	statsPrefix        string
	withRBAC           bool
	trustDomain        string
	trafficTargets     []trafficpolicy.TrafficTargetWithRoutes
	tcpLocalRateLimit  *policyv1alpha1.TCPLocalRateLimitSpec
	tcpGlobalRateLimit *policyv1alpha1.TCPGlobalRateLimitSpec
	hcmBuilder         *httpConnManagerBuilder
	tcpProxyBuilder    *tcpProxyBuilder
}
