package lds

import (
	"fmt"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/utils"
)

// NewResponse creates a new Listener Discovery Response.
// The response build 3 Listeners:
// 1. Inbound listener to handle incoming traffic
// 2. Outbound listener to handle outgoing traffic
// 3. Prometheus listener for metrics
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, cm *certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	var ldsResources []types.Resource

	var statsHeaders map[string]string
	meshConfig := meshCatalog.GetMeshConfig()

	svcList, err := meshCatalog.ListServicesForProxy(proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Str("proxy", proxy.String()).Msgf("Error looking up MeshServices associated with proxy")
		return nil, err
	}

	if meshConfig.Spec.FeatureFlags.EnableWASMStats {
		statsHeaders, err = meshCatalog.GetProxyStatsHeaders(proxy)
		if err != nil {
			log.Err(err).Msgf("Error getting proxy stats headers for proxy %s", proxy)
		}
	}

	// --- OUTBOUND -------------------
	outboundLis := ListenerBuilder().
		Name(OutboundListenerName).
		ProxyIdentity(proxy.Identity).
		Address(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort).
		TrafficDirection(xds_core.TrafficDirection_OUTBOUND).
		PermissiveMesh(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode).
		OutboundMeshTrafficPolicy(meshCatalog.GetOutboundMeshTrafficPolicy(proxy.Identity)).
		ActiveHealthCheck(meshConfig.Spec.FeatureFlags.EnableEnvoyActiveHealthChecks)

	if meshConfig.Spec.Traffic.EnableEgress {
		outboundLis.PermissiveEgress(true)
	} else {
		egressPolicy, err := meshCatalog.GetEgressTrafficPolicy(proxy.Identity)
		if err != nil {
			return nil, fmt.Errorf("error building LDS response: %w", err)
		}
		outboundLis.EgressTrafficPolicy(egressPolicy)
	}
	if meshConfig.Spec.Observability.Tracing.Enable {
		outboundLis.TracingEndpoint(utils.GetTracingEndpoint(meshConfig))
	}
	if meshConfig.Spec.FeatureFlags.EnableWASMStats {
		outboundLis.WASMStatsHeaders(statsHeaders)
	}

	outboundListener, err := outboundLis.Build()
	if err != nil {
		return nil, fmt.Errorf("error building outbound listener for proxy %s: %w", proxy, err)
	}
	if outboundListener == nil {
		// This check is important to prevent attempting to configure a listener without a filter chain which
		// otherwise results in an error.
		log.Debug().Str("proxy", proxy.String()).Msg("Not programming nil outbound listener")
	} else {
		ldsResources = append(ldsResources, outboundListener)
	}

	// --- INBOUND -------------------
	inboundLis := ListenerBuilder().
		Name(InboundListenerName).
		ProxyIdentity(proxy.Identity).
		TrustDomain(cm.GetTrustDomain()).
		Address(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort).
		TrafficDirection(xds_core.TrafficDirection_INBOUND).
		DefaultInboundListenerFilters().
		PermissiveMesh(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode).
		InboundMeshTrafficPolicy(meshCatalog.GetInboundMeshTrafficPolicy(proxy.Identity, svcList)).
		IngressTrafficPolicies(meshCatalog.GetIngressTrafficPolicies(svcList)).
		ActiveHealthCheck(meshConfig.Spec.FeatureFlags.EnableEnvoyActiveHealthChecks).
		SidecarSpec(meshConfig.Spec.Sidecar)

	trafficTargets, err := meshCatalog.ListInboundTrafficTargetsWithRoutes(proxy.Identity)
	if err != nil {
		return nil, fmt.Errorf("error building inbound listener: %w", err)
	}
	inboundLis.TrafficTargets(trafficTargets)

	if meshConfig.Spec.Observability.Tracing.Enable {
		inboundLis.TracingEndpoint(utils.GetTracingEndpoint(meshConfig))
	}
	if extAuthzConfig := utils.ExternalAuthConfigFromMeshConfig(meshConfig); extAuthzConfig.Enable {
		inboundLis.ExtAuthzConfig(&extAuthzConfig)
	}
	if meshConfig.Spec.FeatureFlags.EnableWASMStats {
		inboundLis.WASMStatsHeaders(statsHeaders)
	}

	inboundListener, err := inboundLis.Build()
	if err != nil {
		return nil, fmt.Errorf("error building inbound listener for proxy %s: %w", proxy, err)
	}
	if inboundListener != nil {
		ldsResources = append(ldsResources, inboundListener)
	}

	if enabled, err := meshCatalog.IsMetricsEnabled(proxy); err != nil {
		log.Warn().Str("proxy", proxy.String()).Msgf("Could not find pod for connecting proxy, no metadata was recorded")
	} else if enabled {
		// Build Prometheus listener config
		prometheusConnManager := getPrometheusConnectionManager()
		if prometheusListener, err := buildPrometheusListener(prometheusConnManager); err != nil {
			log.Error().Err(err).Str("proxy", proxy.String()).Msgf("Error building Prometheus listener")
		} else {
			ldsResources = append(ldsResources, prometheusListener)
		}
	}

	return ldsResources, nil
}
