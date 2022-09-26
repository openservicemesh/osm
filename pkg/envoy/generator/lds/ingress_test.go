package lds

import (
	"fmt"
	"testing"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/envoy"

	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestGetIngressFilterChains(t *testing.T) {
	testCases := []struct {
		name                     string
		ingressPolicies          []*trafficpolicy.IngressTrafficPolicy
		expectedFilterChainCount int
	}{
		{
			name: "HTTP ingress",
			ingressPolicies: []*trafficpolicy.IngressTrafficPolicy{
				{
					TrafficMatches: []*trafficpolicy.IngressTrafficMatch{
						{
							Name:           "http-ingress",
							Port:           80,
							Protocol:       "http",
							SourceIPRanges: []string{"10.1.1.0/24"},
						},
					},
				},
			},
			expectedFilterChainCount: 1,
		},
		{
			name: "HTTPS ingress",
			ingressPolicies: []*trafficpolicy.IngressTrafficPolicy{
				{
					TrafficMatches: []*trafficpolicy.IngressTrafficMatch{
						{
							Name:     "https-ingress",
							Port:     80,
							Protocol: "https",
						},
						{
							Name:        "https-ingress_with_sni",
							Port:        80,
							Protocol:    "https",
							ServerNames: []string{"foo.bar.svc.cluster.local"},
						},
					},
				},
			},
			expectedFilterChainCount: 2,
		},
		{
			name:                     "no ingress",
			ingressPolicies:          nil,
			expectedFilterChainCount: 0,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)
			lb := &listenerBuilder{
				proxyIdentity:          tests.BookstoreServiceIdentity,
				ingressTrafficPolicies: tc.ingressPolicies,
				httpTracingEndpoint:    "foo.com/bar",
				extAuthzConfig:         &auth.ExtAuthConfig{Enable: true},
			}

			actual := lb.buildIngressFilterChains()
			assert.Len(actual, tc.expectedFilterChainCount)
		})
	}
}

func TestGetIngressFilterChainFromTrafficMatch(t *testing.T) {
	testCases := []struct {
		name                     string
		trafficMatch             *trafficpolicy.IngressTrafficMatch
		expectedEnvoyFilters     []string
		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectError              bool
	}{
		{
			name: "HTTP traffic match",
			trafficMatch: &trafficpolicy.IngressTrafficMatch{
				Name:     "http-ingress",
				Port:     80,
				Protocol: "http",
			},
			expectedEnvoyFilters: []string{envoy.HTTPConnectionManagerFilterName},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:   &wrapperspb.UInt32Value{Value: 80},
				TransportProtocol: "",
			},
			expectError: false,
		},
		{
			name: "HTTPS traffic match with SNI",
			trafficMatch: &trafficpolicy.IngressTrafficMatch{
				Name:        "https-ingress",
				Port:        80,
				Protocol:    "https",
				ServerNames: []string{"foo.bar.svc.cluster.local"},
			},
			expectedEnvoyFilters: []string{envoy.HTTPConnectionManagerFilterName},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:   &wrapperspb.UInt32Value{Value: 80},
				TransportProtocol: "tls",
				ServerNames:       []string{"foo.bar.svc.cluster.local"},
			},
			expectError: false,
		},
		{
			name: "HTTPS traffic match without SNI",
			trafficMatch: &trafficpolicy.IngressTrafficMatch{
				Name:     "https-ingress",
				Port:     80,
				Protocol: "https",
			},
			expectedEnvoyFilters: []string{envoy.HTTPConnectionManagerFilterName},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:   &wrapperspb.UInt32Value{Value: 80},
				TransportProtocol: "tls",
			},
			expectError: false,
		},
		{
			name: "unsupported protocol",
			trafficMatch: &trafficpolicy.IngressTrafficMatch{
				Name:     "invalid-ingress",
				Port:     80,
				Protocol: "invalid",
			},
			expectedEnvoyFilters:     nil,
			expectedFilterChainMatch: nil,
			expectError:              true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			lb := &listenerBuilder{
				proxyIdentity: tests.BookstoreServiceIdentity,
			}

			actual, err := lb.buildIngressFilterChainFromTrafficMatch(tc.trafficMatch)
			assert.Equal(tc.expectError, err != nil)

			if err == nil {
				assert.Equal(tc.expectedFilterChainMatch, actual.FilterChainMatch)
				assert.Len(actual.Filters, 1) // Single HTTPConnectionManager filter
				assert.Equal(envoy.HTTPConnectionManagerFilterName, actual.Filters[0].Name)
			}
		})
	}
}
