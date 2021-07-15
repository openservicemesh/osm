package lds

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/configurator"
)

func TestGetExtAuthConfig(t *testing.T) {
	testCases := []struct {
		name       string
		authConfig *auth.ExtAuthConfig
		expected   *auth.ExtAuthConfig
	}{
		{
			name: "Ext Auth feature is disabled",
			authConfig: &auth.ExtAuthConfig{
				Enable:           false,
				Address:          "test.xyz",
				Port:             123,
				StatPrefix:       "pref",
				FailureModeAllow: false,
			},
			expected: nil,
		},
		{
			name: "Ext Auth feature is enabled",
			authConfig: &auth.ExtAuthConfig{
				Enable:           true,
				Address:          "test.xyz",
				Port:             123,
				StatPrefix:       "pref",
				FailureModeAllow: false,
			},
			expected: &auth.ExtAuthConfig{
				Enable:           true,
				Address:          "test.xyz",
				Port:             123,
				StatPrefix:       "pref",
				FailureModeAllow: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			lb := &listenerBuilder{
				cfg: mockConfigurator,
			}

			mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(*tc.authConfig).Times(1)

			actual := lb.getExtAuthConfig()
			a.Equal(tc.expected, actual)
		})
	}
}
