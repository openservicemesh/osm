package lds

import (
	"testing"
	"time"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_config_ratelimit "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v3"
	xds_http_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ratelimit/v3"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
)

func TestBuildHTTPGlobalRateLimitFilter(t *testing.T) {
	testCases := []struct {
		name              string
		config            *policyv1alpha1.HTTPGlobalRateLimitSpec
		expectedRateLimit *xds_http_ratelimit.RateLimit
	}{
		{
			name:              "nil config",
			config:            nil,
			expectedRateLimit: nil,
		},
		{
			name: "global rate limit config with all fields set",
			config: &policyv1alpha1.HTTPGlobalRateLimitSpec{
				Domain: "test",
				RateLimitService: policyv1alpha1.RateLimitServiceSpec{
					Host: "foo.bar",
					Port: 8080,
				},
				Timeout:                 &metav1.Duration{Duration: 1 * time.Second},
				FailOpen:                pointer.BoolPtr(false),
				EnableXRateLimitHeaders: pointer.BoolPtr(true),
				ResponseStatusCode:      429,
			},
			expectedRateLimit: &xds_http_ratelimit.RateLimit{
				Domain: "test",
				RateLimitService: &xds_config_ratelimit.RateLimitServiceConfig{
					GrpcService: &xds_core.GrpcService{
						TargetSpecifier: &xds_core.GrpcService_EnvoyGrpc_{
							EnvoyGrpc: &xds_core.GrpcService_EnvoyGrpc{
								ClusterName: "foo.bar|8080",
							},
						},
						Timeout: durationpb.New(1 * time.Second),
					},
					TransportApiVersion: xds_core.ApiVersion_V3,
				},
				Timeout:                 durationpb.New(1 * time.Second),
				FailureModeDeny:         true,
				EnableXRatelimitHeaders: xds_http_ratelimit.RateLimit_DRAFT_VERSION_03,
				RateLimitedStatus:       &xds_type.HttpStatus{Code: xds_type.StatusCode(429)},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			filter := buildHTTPGlobalRateLimitFilter(tc.config)
			if tc.expectedRateLimit == nil {
				a.Nil(filter)
				return
			}

			rateLimit := &xds_http_ratelimit.RateLimit{}
			err := filter.GetTypedConfig().UnmarshalTo(rateLimit)
			a.Nil(err)

			if diff := cmp.Diff(tc.expectedRateLimit, rateLimit, protocmp.Transform()); diff != "" {
				t.Errorf("RateLimit mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
