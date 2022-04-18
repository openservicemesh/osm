package lds

import (
	"testing"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestGetEgressHTTPFilterChain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	testCases := []struct {
		name                     string
		destinationPort          int
		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectError              bool
	}{
		{
			name:            "egress HTTP filter chain for port 80",
			destinationPort: 80,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 80},
				ApplicationProtocols: []string{"http/1.0", "http/1.1", "h2c"},
			},
			expectError: false,
		},
		{
			name:            "egress HTTP filter chain for port 100",
			destinationPort: 100,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 100},
				ApplicationProtocols: []string{"http/1.0", "http/1.1", "h2c"},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			lb := &listenerBuilder{
				cfg: mockConfigurator,
			}
			mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
			mockConfigurator.EXPECT().GetTracingEndpoint().Return("some-endpoint").AnyTimes()
			mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
				EnableEgressPolicy: true,
				EnableWASMStats:    false}).AnyTimes()
			actual, err := lb.getEgressHTTPFilterChain(tc.destinationPort)

			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedFilterChainMatch, actual.FilterChainMatch)
			assert.Len(actual.Filters, 1) // Single HTTPConnectionManager filter
			assert.Equal(wellknown.HTTPConnectionManager, actual.Filters[0].Name)
		})
	}
}

func TestGetEgressTCPFilterChain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	testCases := []struct {
		name                     string
		trafficMatch             trafficpolicy.TrafficMatch
		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectError              bool
	}{
		{
			name: "egress TCP filter chain for port match",
			trafficMatch: trafficpolicy.TrafficMatch{
				DestinationPort:     80,
				DestinationProtocol: "tcp",
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort: &wrapperspb.UInt32Value{Value: 80},
			},
			expectError: false,
		},
		{
			name: "egress TCP filter chain for port and IP ranges match",
			trafficMatch: trafficpolicy.TrafficMatch{
				DestinationPort:     100,
				DestinationProtocol: "tcp",
				DestinationIPRanges: []string{"10.0.0.0/24", "8.8.8.8/32"},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort: &wrapperspb.UInt32Value{Value: 100},
				PrefixRanges: []*xds_core.CidrRange{
					{
						AddressPrefix: "10.0.0.0",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 24,
						},
					},
					{
						AddressPrefix: "8.8.8.8",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "egress TCP filter chain for port, IP ranges and SNI match",
			trafficMatch: trafficpolicy.TrafficMatch{
				DestinationPort:     100,
				DestinationProtocol: "tcp",
				DestinationIPRanges: []string{"10.0.0.0/24", "8.8.8.8/32"},
				ServerNames:         []string{"foo.com"},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort: &wrapperspb.UInt32Value{Value: 100},
				PrefixRanges: []*xds_core.CidrRange{
					{
						AddressPrefix: "10.0.0.0",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 24,
						},
					},
					{
						AddressPrefix: "8.8.8.8",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
				},
				ServerNames: []string{"foo.com"},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
				EnableEgressPolicy: true,
				EnableWASMStats:    false,
			}).AnyTimes()

			lb := &listenerBuilder{}

			actual, err := lb.getEgressTCPFilterChain(tc.trafficMatch)

			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedFilterChainMatch, actual.FilterChainMatch)
			assert.Len(actual.Filters, 1) // Single TCPProxy filter
			assert.Equal(wellknown.TCPProxy, actual.Filters[0].Name)
		})
	}
}

func TestGetEgressFilterChainsForMatches(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	testCases := []struct {
		name                     string
		trafficMatches           []*trafficpolicy.TrafficMatch
		expectedFilterChainCount int
	}{
		{
			name: "Multiple valid traffic matches",
			trafficMatches: []*trafficpolicy.TrafficMatch{
				{
					DestinationPort:     100,
					DestinationProtocol: "http",
					ServerNames:         []string{"foo.com"},
				},
				{
					DestinationPort:     100,
					DestinationProtocol: "https",
					DestinationIPRanges: []string{"10.0.0.0/24", "8.8.8.8/32"},
					ServerNames:         []string{"foo.com"},
				},
				{
					DestinationPort:     100,
					DestinationProtocol: "tcp",
					DestinationIPRanges: []string{"10.0.0.0/24", "8.8.8.8/32"},
				},
			},
			expectedFilterChainCount: 3, // 1 for each match
		},
		{
			name: "Invalid traffic matches should be ignored",
			trafficMatches: []*trafficpolicy.TrafficMatch{
				{
					DestinationPort:     100,
					DestinationProtocol: "http",
					ServerNames:         []string{"foo.com"},
				},
				{
					DestinationPort:     100,
					DestinationProtocol: "invalid",
				},
			},
			expectedFilterChainCount: 1, // 1 for the valid match, match with invalid protocol is ignored
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			lb := &listenerBuilder{
				cfg: mockConfigurator,
			}
			mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
			mockConfigurator.EXPECT().GetTracingEndpoint().Return("some-endpoint").AnyTimes()
			mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
				EnableEgressPolicy: true,
				EnableWASMStats:    false,
			}).AnyTimes()

			actual := lb.getEgressFilterChainsForMatches(tc.trafficMatches)

			assert.Len(actual, tc.expectedFilterChainCount)
		})
	}
}
