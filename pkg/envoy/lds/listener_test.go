package lds

import (
	"testing"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestBuildPrometheusListener(t *testing.T) {
	a := assert.New(t)

	connManager := getPrometheusConnectionManager()
	listener, err := buildPrometheusListener(connManager)
	a.NotNil(listener)
	a.Nil(err)
}

func TestGetFilterMatchPredicateForPorts(t *testing.T) {
	testCases := []struct {
		name          string
		ports         []int
		expectedMatch *xds_listener.ListenerFilterChainMatchPredicate
	}{
		{
			name:  "single port to exclude",
			ports: []int{80},
			expectedMatch: &xds_listener.ListenerFilterChainMatchPredicate{
				Rule: &xds_listener.ListenerFilterChainMatchPredicate_DestinationPortRange{
					DestinationPortRange: &xds_type.Int32Range{
						Start: 80, // Start is inclusive
						End:   81, // End is exclusive
					},
				},
			},
		},
		{
			name:  "multiple ports to exclude",
			ports: []int{80, 90},
			expectedMatch: &xds_listener.ListenerFilterChainMatchPredicate{
				Rule: &xds_listener.ListenerFilterChainMatchPredicate_OrMatch{
					OrMatch: &xds_listener.ListenerFilterChainMatchPredicate_MatchSet{
						Rules: []*xds_listener.ListenerFilterChainMatchPredicate{
							{
								Rule: &xds_listener.ListenerFilterChainMatchPredicate_DestinationPortRange{
									DestinationPortRange: &xds_type.Int32Range{
										Start: 80, // Start is inclusive
										End:   81, // End is exclusive
									},
								},
							},
							{
								Rule: &xds_listener.ListenerFilterChainMatchPredicate_DestinationPortRange{
									DestinationPortRange: &xds_type.Int32Range{
										Start: 90, // Start is inclusive
										End:   91, // End is exclusive
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:          "no ports specified",
			ports:         nil,
			expectedMatch: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual := getFilterMatchPredicateForPorts(tc.ports)
			a.Equal(tc.expectedMatch, actual)
		})
	}
}

func TestGetFilterMatchPredicateForTrafficMatches(t *testing.T) {
	testCases := []struct {
		name          string
		matches       []*trafficpolicy.TrafficMatch
		expectedMatch *xds_listener.ListenerFilterChainMatchPredicate
	}{
		{
			name: "no server-first ports",
			matches: []*trafficpolicy.TrafficMatch{
				{
					DestinationProtocol: "tcp",
					DestinationPort:     80,
				},
			},
			expectedMatch: nil,
		},
		{
			name: "server-first port present",
			matches: []*trafficpolicy.TrafficMatch{
				{
					DestinationProtocol: "tcp",
					DestinationPort:     80,
				},
				{
					DestinationProtocol: "tcp-server-first",
					DestinationPort:     100,
				},
			},
			expectedMatch: &xds_listener.ListenerFilterChainMatchPredicate{
				Rule: &xds_listener.ListenerFilterChainMatchPredicate_DestinationPortRange{
					DestinationPortRange: &xds_type.Int32Range{
						Start: 100, // Start is inclusive
						End:   101, // End is exclusive
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual := getFilterMatchPredicateForTrafficMatches(tc.matches)
			a.Equal(tc.expectedMatch, actual)
		})
	}
}
