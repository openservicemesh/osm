package lds

import (
	"fmt"
	"time"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_config_ratelimit "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v3"
	xds_common_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	xds_http_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ratelimit/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_network_local_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/local_ratelimit/v3"
	xds_network_global_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/ratelimit/v3"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/protobuf"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

func buildTCPLocalRateLimitFilter(config *policyv1alpha1.TCPLocalRateLimitSpec, statPrefix string) (*xds_listener.Filter, error) {
	if config == nil {
		return nil, nil
	}

	var fillInterval time.Duration
	switch config.Unit {
	case "second":
		fillInterval = time.Second
	case "minute":
		fillInterval = time.Minute
	case "hour":
		fillInterval = time.Hour
	default:
		return nil, fmt.Errorf("invalid unit %q for TCP connection rate limiting", config.Unit)
	}

	rateLimit := &xds_network_local_ratelimit.LocalRateLimit{
		StatPrefix: statPrefix,
		TokenBucket: &xds_type.TokenBucket{
			MaxTokens:     config.Connections + config.Burst,
			TokensPerFill: wrapperspb.UInt32(config.Connections),
			FillInterval:  durationpb.New(fillInterval),
		},
	}

	marshalledConfig, err := anypb.New(rateLimit)
	if err != nil {
		return nil, err
	}

	filter := &xds_listener.Filter{
		Name:       envoy.L4LocalRateLimitFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledConfig},
	}

	return filter, nil
}

func buildTCPGlobalRateLimitFilter(config *policyv1alpha1.TCPGlobalRateLimitSpec, statPrefix string) (*xds_listener.Filter, error) {
	if config == nil {
		return nil, nil
	}

	rateLimit := &xds_network_global_ratelimit.RateLimit{
		StatPrefix: statPrefix,
		Domain:     config.Domain,
		RateLimitService: &xds_config_ratelimit.RateLimitServiceConfig{
			GrpcService: &xds_core.GrpcService{
				TargetSpecifier: &xds_core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &xds_core.GrpcService_EnvoyGrpc{
						ClusterName: service.RateLimitServiceClusterName(config.RateLimitService),
					},
				},
			},
			TransportApiVersion: xds_core.ApiVersion_V3,
		},
	}

	var descriptors []*xds_common_ratelimit.RateLimitDescriptor
	for _, desc := range config.Descriptors {
		var entries []*xds_common_ratelimit.RateLimitDescriptor_Entry
		for _, entry := range desc.Entries {
			entries = append(entries, &xds_common_ratelimit.RateLimitDescriptor_Entry{Key: entry.Key, Value: entry.Value})
		}

		descriptors = append(descriptors, &xds_common_ratelimit.RateLimitDescriptor{Entries: entries})
	}
	rateLimit.Descriptors = descriptors

	if config.Timeout != nil {
		rateLimit.Timeout = durationpb.New(config.Timeout.Duration)
		rateLimit.RateLimitService.GrpcService.Timeout = durationpb.New(config.Timeout.Duration)
	}

	if config.FailOpen != nil {
		rateLimit.FailureModeDeny = !*config.FailOpen
	}

	marshalledConfig, err := anypb.New(rateLimit)
	if err != nil {
		return nil, err
	}

	filter := &xds_listener.Filter{
		Name:       envoy.L4GlobalRateLimitFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledConfig},
	}

	return filter, nil
}

func buildHTTPGlobalRateLimitFilter(config *policyv1alpha1.HTTPGlobalRateLimitSpec) *xds_hcm.HttpFilter {
	if config == nil {
		return nil
	}

	rateLimit := &xds_http_ratelimit.RateLimit{
		Domain: config.Domain,
		RateLimitService: &xds_config_ratelimit.RateLimitServiceConfig{
			GrpcService: &xds_core.GrpcService{
				TargetSpecifier: &xds_core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &xds_core.GrpcService_EnvoyGrpc{
						ClusterName: service.RateLimitServiceClusterName(config.RateLimitService),
					},
				},
			},
			TransportApiVersion: xds_core.ApiVersion_V3,
		},
		EnableXRatelimitHeaders: xds_http_ratelimit.RateLimit_OFF,
	}

	if config.Timeout != nil {
		rateLimit.Timeout = durationpb.New(config.Timeout.Duration)
		rateLimit.RateLimitService.GrpcService.Timeout = durationpb.New(config.Timeout.Duration)
	}

	if config.FailOpen != nil {
		rateLimit.FailureModeDeny = !*config.FailOpen
	}

	if config.EnableXRateLimitHeaders != nil && *config.EnableXRateLimitHeaders {
		rateLimit.EnableXRatelimitHeaders = xds_http_ratelimit.RateLimit_DRAFT_VERSION_03
	}

	if config.ResponseStatusCode > 0 {
		rateLimit.RateLimitedStatus = &xds_type.HttpStatus{Code: xds_type.StatusCode(config.ResponseStatusCode)}
	}

	return &xds_hcm.HttpFilter{
		Name: envoy.HTTPGlobalRateLimitFilterName,
		ConfigType: &xds_hcm.HttpFilter_TypedConfig{
			TypedConfig: protobuf.MustMarshalAny(rateLimit),
		},
	}
}
