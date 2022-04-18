package lds

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/configurator"
)

func TestGetWASMStatsHeaders(t *testing.T) {
	testCases := []struct {
		name         string
		enabled      bool
		statsHeaders map[string]string
		expected     map[string]string
	}{
		{
			name:         "WASM feature is disabled",
			enabled:      false,
			statsHeaders: map[string]string{"k1": "v1"},
			expected:     nil,
		},
		{
			name:         "WASM feature is enabled",
			enabled:      true,
			statsHeaders: map[string]string{"k1": "v1"},
			expected:     map[string]string{"k1": "v1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			lb := &listenerBuilder{
				cfg:          mockConfigurator,
				statsHeaders: tc.statsHeaders,
			}

			mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableWASMStats: tc.enabled}).Times(1)

			actual := lb.getWASMStatsHeaders()
			a.Equal(tc.expected, actual)
		})
	}
}
