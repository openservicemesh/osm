package lds

import (
	"testing"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// Tests TestGetFilterForService checks that a proper filter type is properly returned
// for given config parameters and service

var _ = Describe("Construct inbound listeners", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingHost().Return(constants.DefaultTracingHost).AnyTimes()
	mockConfigurator.EXPECT().GetTracingPort().Return(constants.DefaultTracingPort).AnyTimes()

	Context("Test creation of inbound listener", func() {
		It("Tests the inbound listener config", func() {
			listener := newInboundListener()
			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort)))
			Expect(listener.AccessLog).NotTo(BeEmpty())
			Expect(len(listener.ListenerFilters)).To(Equal(2)) // TlsInspector, OriginalDestination listener filter
			Expect(listener.ListenerFilters[0].Name).To(Equal(wellknown.TlsInspector))
			Expect(listener.TrafficDirection).To(Equal(xds_core.TrafficDirection_INBOUND))
		})
	})

	Context("Test creation of Prometheus listener", func() {
		It("Tests the Prometheus listener config", func() {
			connManager := getPrometheusConnectionManager()
			listener, _ := buildPrometheusListener(connManager)
			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyPrometheusInboundListenerPort)))
			Expect(len(listener.ListenerFilters)).To(Equal(0)) //  no listener filters
			Expect(listener.TrafficDirection).To(Equal(xds_core.TrafficDirection_INBOUND))
		})
	})
})

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
			assert := tassert.New(t)

			actual := getFilterMatchPredicateForPorts(tc.ports)
			assert.Equal(tc.expectedMatch, actual)
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
			assert := tassert.New(t)

			actual := getFilterMatchPredicateForTrafficMatches(tc.matches)
			assert.Equal(tc.expectedMatch, actual)
		})
	}
}

func TestNewOutboundListener(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	identity := identity.K8sServiceAccount{}.ToServiceIdentity()
	meshCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	meshCatalog.EXPECT().GetEgressTrafficPolicy(gomock.Any()).Return(nil, nil).Times(1)
	meshCatalog.EXPECT().GetOutboundMeshTrafficPolicy(identity).Return(&trafficpolicy.OutboundMeshTrafficPolicy{
		TrafficMatches: []*trafficpolicy.TrafficMatch{
			{
				WeightedClusters: []service.WeightedCluster{{}},
				DestinationIPRanges: []string{
					"0.0.0.0/0",
				},
				DestinationPort:     1,
				DestinationProtocol: constants.ProtocolTCPServerFirst,
			},
		},
	}).Times(2)
	cfg := configurator.NewMockConfigurator(mockCtrl)
	cfg.EXPECT().IsEgressEnabled().Return(false).Times(1)
	cfg.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
		EnableEgressPolicy: true,
	}).Times(1)

	lb := newListenerBuilder(meshCatalog, identity, cfg, nil)

	assert := tassert.New(t)
	listener, err := lb.newOutboundListener()
	assert.NoError(err)

	assert.Len(listener.ListenerFilters, 3) // OriginalDst, TlsInspector, HttpInspector
	assert.NotEmpty(listener.AccessLog)
	assert.Equal(wellknown.TlsInspector, listener.ListenerFilters[1].Name)
	assert.Equal(&xds_listener.ListenerFilterChainMatchPredicate{
		Rule: &xds_listener.ListenerFilterChainMatchPredicate_DestinationPortRange{
			DestinationPortRange: &xds_type.Int32Range{
				Start: 1,
				End:   2,
			},
		},
	}, listener.ListenerFilters[1].FilterDisabled)
	assert.Equal(wellknown.HttpInspector, listener.ListenerFilters[2].Name)
	assert.Equal(listener.ListenerFilters[1].FilterDisabled, listener.ListenerFilters[2].FilterDisabled)
}
