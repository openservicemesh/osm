package lds

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetIngressFilterChains(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proxyService := tests.BookstoreV1Service

	testCases := []struct {
		name                 string
		httpsIngress         bool // true for https, false for http
		svcPortToProtocolMap map[uint32]string
		portToProtocolErr    error // error to return if port:protocol mapping returns an error

		expectedFilterChainCount               int
		expectedFilterNamesPerFilterChain      []string
		expectedFilterChainMatchPerFilterChain []*xds_listener.FilterChainMatch
	}{
		{
			// Test case 1
			name:                 "HTTP ingress filter chain for service with multiple ports",
			httpsIngress:         false,
			svcPortToProtocolMap: map[uint32]string{80: "http", 90: "http"},
			portToProtocolErr:    nil,

			expectedFilterChainCount:          2, // 1 per http port on the service
			expectedFilterNamesPerFilterChain: []string{wellknown.HTTPConnectionManager},
			expectedFilterChainMatchPerFilterChain: []*xds_listener.FilterChainMatch{
				{
					DestinationPort:   &wrapperspb.UInt32Value{Value: 80},
					TransportProtocol: "",
				},
				{
					DestinationPort:   &wrapperspb.UInt32Value{Value: 90},
					TransportProtocol: "",
				},
			},
		},

		{
			// Test case 2
			name:                 "HTTPS ingress filter chain for service with multiple ports",
			httpsIngress:         true,
			svcPortToProtocolMap: map[uint32]string{80: "http", 90: "http"},
			portToProtocolErr:    nil,

			expectedFilterChainCount:          4, // number of ports * 2; 2 because for HTTPS 2 filter chains are created: with and without SNI matching
			expectedFilterNamesPerFilterChain: []string{wellknown.HTTPConnectionManager},
			expectedFilterChainMatchPerFilterChain: []*xds_listener.FilterChainMatch{
				{
					DestinationPort:   &wrapperspb.UInt32Value{Value: 80},
					TransportProtocol: "tls",
				},
				{
					DestinationPort:   &wrapperspb.UInt32Value{Value: 90},
					TransportProtocol: "tls",
				},
				{
					DestinationPort:   &wrapperspb.UInt32Value{Value: 80},
					TransportProtocol: "tls",
					ServerNames:       []string{proxyService.ServerName()},
				},
				{
					DestinationPort:   &wrapperspb.UInt32Value{Value: 90},
					TransportProtocol: "tls",
					ServerNames:       []string{proxyService.ServerName()},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

			lb := &listenerBuilder{
				meshCatalog: mockCatalog,
				cfg:         mockConfigurator,
				svcAccount:  tests.BookstoreServiceAccount,
			}

			// Mock catalog call to get port:protocol mapping for service
			mockCatalog.EXPECT().GetTargetPortToProtocolMappingForService(proxyService).Return(tc.svcPortToProtocolMap, tc.portToProtocolErr).Times(1)
			// Mock configurator calls to determine HTTP vs HTTPS ingress
			mockConfigurator.EXPECT().UseHTTPSIngress().Return(tc.httpsIngress).AnyTimes()
			// Mock calls used to build the HTTP connection manager
			mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()

			filterChains := lb.getIngressFilterChains(proxyService)

			// Test the filter chains
			assert.Len(filterChains, tc.expectedFilterChainCount)

			var actualFilterChainMatchPerFilterChain []*xds_listener.FilterChainMatch
			for _, filterChain := range filterChains {
				assert.Len(filterChain.Filters, len(tc.expectedFilterNamesPerFilterChain))
				for i, filter := range filterChain.Filters {
					assert.Equal(tc.expectedFilterNamesPerFilterChain[i], filter.Name)
				}
				actualFilterChainMatchPerFilterChain = append(actualFilterChainMatchPerFilterChain, filterChain.FilterChainMatch)
			}

			assert.ElementsMatch(tc.expectedFilterChainMatchPerFilterChain, actualFilterChainMatchPerFilterChain)
		})
	}
}

func TestGetIngressTransportProtocol(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                      string
		forHTTPS                  bool
		expectedTransportProtocol string
	}{
		{
			name:                      "for HTTP",
			forHTTPS:                  false,
			expectedTransportProtocol: "",
		},
		{
			name:                      "for HTTPS",
			forHTTPS:                  true,
			expectedTransportProtocol: "tls",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			actual := getIngressTransportProtocol(tc.forHTTPS)
			assert.Equal(tc.expectedTransportProtocol, actual)
		})
	}
}
