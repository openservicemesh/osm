package lds

import (
	"fmt"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestGetOutboundHTTPFilterChainForService(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	// Mock calls used to build the HTTP connection manager
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-api").AnyTimes()

	lb := &listenerBuilder{
		meshCatalog: mockCatalog,
		cfg:         mockConfigurator,
		svcAccount:  tests.BookbuyerServiceAccount,
	}

	testCases := []struct {
		name              string
		expectedEndpoints []endpoint.Endpoint
		expectError       bool
	}{
		{
			name: "service with multiple endpoints",
			expectedEndpoints: []endpoint.Endpoint{
				{IP: net.ParseIP("1.1.1.1"), Port: 80},
				{IP: net.ParseIP("2.2.2.2"), Port: 80},
			},
			expectError: false,
		},
		{
			name:              "service with no endpoints",
			expectedEndpoints: []endpoint.Endpoint{},
			expectError:       true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockCatalog.EXPECT().GetResolvableServiceEndpoints(tests.BookstoreApexService).Return(tc.expectedEndpoints, nil)
			httpFilterChain, err := lb.getOutboundHTTPFilterChainForService(tests.BookstoreApexService)

			assert.Equal(err != nil, tc.expectError)

			if err != nil {
				assert.Nil(httpFilterChain)
			} else {
				assert.NotNil(httpFilterChain)
				assert.Len(httpFilterChain.FilterChainMatch.PrefixRanges, len(tc.expectedEndpoints))
			}
		})
	}
}

func TestGetInboundMeshHTTPFilterChain(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	// Mock calls used to build the HTTP connection manager
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-api").AnyTimes()

	lb := &listenerBuilder{
		meshCatalog: mockCatalog,
		cfg:         mockConfigurator,
		svcAccount:  tests.BookbuyerServiceAccount,
	}

	proxyService := tests.BookbuyerService

	testCases := []struct {
		name           string
		permissiveMode bool
		port           uint32

		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectedFilterNames      []string
		expectError              bool
	}{
		{
			name:           "inbound HTTP filter chain with permissive mode disabled",
			permissiveMode: false,
			port:           80,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 80},
				ServerNames:          []string{proxyService.ServerName()},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{wellknown.RoleBasedAccessControl, wellknown.HTTPConnectionManager},
			expectError:         false,
		},

		{
			name:           "inbound HTTP filter chain with permissive mode enabled",
			permissiveMode: true,
			port:           90,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{proxyService.ServerName()},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{wellknown.HTTPConnectionManager},
			expectError:         false,
		},
	}

	trafficTargets := []trafficpolicy.TrafficTargetWithRoutes{
		{
			Name:        "ns-1/test-1",
			Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
			Sources: []identity.ServiceIdentity{
				identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
				identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
			},
			TCPRouteMatches: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).Times(1)
			if !tc.permissiveMode {
				// mock catalog calls used to build the RBAC filter
				mockCatalog.EXPECT().ListInboundTrafficTargetsWithRoutes(lb.svcAccount).Return(trafficTargets, nil).Times(1)
			}

			filterChain, err := lb.getInboundMeshHTTPFilterChain(proxyService, tc.port)

			assert.Equal(err != nil, tc.expectError)
			assert.Equal(filterChain.FilterChainMatch, tc.expectedFilterChainMatch)
			assert.Len(filterChain.Filters, len(tc.expectedFilterNames))
			for i, filter := range filterChain.Filters {
				assert.Equal(filter.Name, tc.expectedFilterNames[i])
			}
		})
	}
}

func TestGetInboundMeshTCPFilterChain(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	// Mock calls used to build the HTTP connection manager
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-api").AnyTimes()

	lb := &listenerBuilder{
		meshCatalog: mockCatalog,
		cfg:         mockConfigurator,
		svcAccount:  tests.BookbuyerServiceAccount,
	}

	proxyService := tests.BookbuyerService

	testCases := []struct {
		name           string
		permissiveMode bool
		port           uint32

		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectedFilterNames      []string
		expectError              bool
	}{
		{
			name:           "inbound TCP filter chain with permissive mode disabled",
			permissiveMode: false,
			port:           80,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 80},
				ServerNames:          []string{proxyService.ServerName()},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{wellknown.RoleBasedAccessControl, wellknown.TCPProxy},
			expectError:         false,
		},

		{
			name:           "inbound TCP filter chain with permissive mode enabled",
			permissiveMode: true,
			port:           90,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{proxyService.ServerName()},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{wellknown.TCPProxy},
			expectError:         false,
		},
	}

	trafficTargets := []trafficpolicy.TrafficTargetWithRoutes{
		{
			Name:        "ns-1/test-1",
			Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
			Sources: []identity.ServiceIdentity{
				identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
				identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
			},
			TCPRouteMatches: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).Times(1)
			if !tc.permissiveMode {
				// mock catalog calls used to build the RBAC filter
				mockCatalog.EXPECT().ListInboundTrafficTargetsWithRoutes(lb.svcAccount).Return(trafficTargets, nil).Times(1)
			}

			filterChain, err := lb.getInboundMeshTCPFilterChain(proxyService, tc.port)

			assert.Equal(err != nil, tc.expectError)
			assert.Equal(filterChain.FilterChainMatch, tc.expectedFilterChainMatch)
			assert.Len(filterChain.Filters, len(tc.expectedFilterNames))
			for i, filter := range filterChain.Filters {
				assert.Equal(filter.Name, tc.expectedFilterNames[i])
			}
		})
	}
}

// Tests getOutboundFilterChainMatchForService and ensures the filter chain match returned is as expected
func TestGetOutboundFilterChainMatchForService(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)

	lb := newListenerBuilder(mockCatalog, tests.BookbuyerServiceAccount, mockConfigurator)

	testCases := []struct {
		name        string
		endpoints   []endpoint.Endpoint
		appProtocol string

		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectError              bool
	}{
		{
			// test case 1
			name: "outbound HTTP filter chain for service with endpoints",
			endpoints: []endpoint.Endpoint{
				{
					IP: net.IPv4(192, 168, 10, 1),
				},
				{
					IP: net.IPv4(192, 168, 20, 2),
				},
			},
			appProtocol: httpAppProtocol,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				// HTTP filter chain should only match on supported HTTP protocols that the downstream can use
				// to originate a request.
				ApplicationProtocols: []string{"http/1.0", "http/1.1", "h2c"},
				PrefixRanges: []*xds_core.CidrRange{
					{
						AddressPrefix: "192.168.10.1",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
					{
						AddressPrefix: "192.168.20.2",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
				},
			},
			expectError: false,
		},

		{
			// test case 2
			name:                     "outbound HTTP filter chain for service without endpoints",
			endpoints:                []endpoint.Endpoint{},
			appProtocol:              httpAppProtocol,
			expectedFilterChainMatch: nil,
			expectError:              true,
		},

		{
			// test case 3
			name: "outbound TCP filter chain for service with endpoints",
			endpoints: []endpoint.Endpoint{
				{
					IP: net.IPv4(192, 168, 10, 1),
				},
				{
					IP: net.IPv4(192, 168, 20, 2),
				},
			},
			appProtocol: tcpAppProtocol,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				PrefixRanges: []*xds_core.CidrRange{
					{
						AddressPrefix: "192.168.10.1",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
					{
						AddressPrefix: "192.168.20.2",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
				},
			},
			expectError: false,
		},

		{
			// test case 4
			name:                     "outbound TCP filter chain for service without endpoints",
			endpoints:                []endpoint.Endpoint{},
			appProtocol:              httpAppProtocol,
			expectedFilterChainMatch: nil,
			expectError:              true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			// mock endpoints returned
			mockCatalog.EXPECT().GetResolvableServiceEndpoints(tests.BookstoreApexService).Return(tc.endpoints, nil)

			filterChainMatch, err := lb.getOutboundFilterChainMatchForService(tests.BookstoreApexService, tc.appProtocol)
			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedFilterChainMatch, filterChainMatch)
		})
	}
}
