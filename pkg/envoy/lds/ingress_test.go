package lds

import (
	"fmt"
	"testing"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestGetIngressFilterChains(t *testing.T) {
	testCases := []struct {
		name                     string
		ingressPolicy            *trafficpolicy.IngressTrafficPolicy
		expectedFilterChainCount int
	}{
		{
			name: "HTTP ingress",
			ingressPolicy: &trafficpolicy.IngressTrafficPolicy{
				TrafficMatches: []*trafficpolicy.IngressTrafficMatch{
					{
						Name:     "http-ingress",
						Port:     80,
						Protocol: "http",
					},
				},
			},
			expectedFilterChainCount: 1,
		},
		{
			name: "HTTPS ingress",
			ingressPolicy: &trafficpolicy.IngressTrafficPolicy{
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
			expectedFilterChainCount: 2,
		},
		{
			name:                     "no ingress",
			ingressPolicy:            nil,
			expectedFilterChainCount: 0,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)

			lb := &listenerBuilder{
				serviceIdentity: tests.BookstoreServiceIdentity,
				cfg:             mockConfigurator,
				meshCatalog:     mockCatalog,
			}

			testSvc := tests.BookstoreV1Service

			mockCatalog.EXPECT().GetIngressTrafficPolicy(testSvc).Return(tc.ingressPolicy, nil)
			mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
			mockConfigurator.EXPECT().GetTracingEndpoint().Return("test").AnyTimes()
			mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
				Enable: false,
			}).AnyTimes()
			mockConfigurator.EXPECT().GetMeshConfig().AnyTimes()

			actual := lb.getIngressFilterChains(testSvc)
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
			expectedEnvoyFilters: []string{wellknown.HTTPConnectionManager},
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
			expectedEnvoyFilters: []string{wellknown.HTTPConnectionManager},
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
			expectedEnvoyFilters: []string{wellknown.HTTPConnectionManager},
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
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

			lb := &listenerBuilder{
				serviceIdentity: tests.BookstoreServiceIdentity,
				cfg:             mockConfigurator,
			}

			mockConfigurator.EXPECT().IsTracingEnabled().Return(false)
			mockConfigurator.EXPECT().GetTracingEndpoint().Return("test")
			mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
				Enable: false,
			})

			actual, err := lb.getIngressFilterChainFromTrafficMatch(tc.trafficMatch, configv1alpha3.SidecarSpec{})
			assert.Equal(tc.expectError, err != nil)

			if err == nil {
				assert.Equal(tc.expectedFilterChainMatch, actual.FilterChainMatch)
				assert.Len(actual.Filters, 1) // Single HTTPConnectionManager filter
				assert.Equal(wellknown.HTTPConnectionManager, actual.Filters[0].Name)
			}
		})
	}
}
