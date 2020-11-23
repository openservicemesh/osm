package lds

import (
	"fmt"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/tests"
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
