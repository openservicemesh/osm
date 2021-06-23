package lds

import (
	"testing"
	"time"

	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/auth"
)

func TestHTTPConnbuild(t *testing.T) {
	notContains := func(filters []*xds_hcm.HttpFilter, filterName string) bool {
		for _, f := range filters {
			if f.Name == filterName {
				return false
			}
		}
		return true
	}
	contains := func(filters []*xds_hcm.HttpFilter, filterName string) bool {
		return !notContains(filters, filterName)
	}

	testCases := []struct {
		name       string
		option     httpConnManagerOptions
		assertFunc func(*assert.Assertions, *xds_hcm.HttpConnectionManager)
	}{
		{
			name: "stat prefix",
			option: httpConnManagerOptions{
				rdsRoutConfigName: "something",
			},
			assertFunc: func(a *assert.Assertions, connManager *xds_hcm.HttpConnectionManager) {
				a.Equal("mesh-http-conn-manager.something", connManager.StatPrefix)
			},
		},
		{
			name: "tracing config when tracing is enabled",
			option: httpConnManagerOptions{
				enableTracing:      true,
				tracingAPIEndpoint: "/api/v1/trace",
			},
			assertFunc: func(a *assert.Assertions, connManager *xds_hcm.HttpConnectionManager) {
				a.NotNil(connManager.Tracing)
				a.True(connManager.Tracing.Verbose)
				a.Equal("envoy.tracers.zipkin", connManager.Tracing.Provider.Name)
			},
		},
		{
			name: "tracing config when tracing is disabled",
			option: httpConnManagerOptions{
				enableTracing: false,
			},
			assertFunc: func(a *assert.Assertions, connManager *xds_hcm.HttpConnectionManager) {
				a.Nil(connManager.Tracing)
			},
		},
		{
			name: "WASM config when WASM stats headers are unset",
			option: httpConnManagerOptions{
				wasmStatsHeaders: nil,
			},
			assertFunc: func(a *assert.Assertions, connManager *xds_hcm.HttpConnectionManager) {
				a.Nil(connManager.LocalReplyConfig)
				a.True(notContains(connManager.HttpFilters, wellknown.Lua))
				a.True(notContains(connManager.HttpFilters, "envoy.filters.http.wasm"))
			},
		},
		{
			name: "WASM config when WASM stats headers are set",
			option: httpConnManagerOptions{
				wasmStatsHeaders: map[string]string{"k1": "v1"},
			},
			assertFunc: func(a *assert.Assertions, connManager *xds_hcm.HttpConnectionManager) {
				a.NotNil(connManager.LocalReplyConfig)
				a.Equal("unknown", connManager.GetLocalReplyConfig().GetMappers()[0].HeadersToAdd[0].Header.Value)
				a.True(contains(connManager.HttpFilters, wellknown.Lua))
				a.True(contains(connManager.HttpFilters, "envoy.filters.http.wasm"))
			},
		},
		{
			name: "External auth config when set is enabled for inbound",
			option: httpConnManagerOptions{
				direction: inbound,
				extAuthConfig: &auth.ExtAuthConfig{
					Enable:           true,
					Address:          "test.xyz",
					Port:             123,
					StatPrefix:       "pref",
					AuthzTimeout:     3 * time.Second,
					FailureModeAllow: false,
				},
			},
			assertFunc: func(a *assert.Assertions, connManager *xds_hcm.HttpConnectionManager) {
				a.True(contains(connManager.HttpFilters, wellknown.HTTPExternalAuthorization))
			},
		},
		{
			name: "External auth config when set is disabled for outbound",
			option: httpConnManagerOptions{
				direction: outbound,
				extAuthConfig: &auth.ExtAuthConfig{
					Enable: true,
				},
			},
			assertFunc: func(a *assert.Assertions, connManager *xds_hcm.HttpConnectionManager) {
				a.True(notContains(connManager.HttpFilters, wellknown.HTTPExternalAuthorization))
			},
		},
		{
			name: "health check config present when enabled",
			option: httpConnManagerOptions{
				enableActiveHealthChecks: true,
			},
			assertFunc: func(a *assert.Assertions, connManager *xds_hcm.HttpConnectionManager) {
				a.True(contains(connManager.HttpFilters, wellknown.HealthCheck))
			},
		},
		{
			name: "health check config absent when disabled",
			option: httpConnManagerOptions{
				enableActiveHealthChecks: false,
			},
			assertFunc: func(a *assert.Assertions, connManager *xds_hcm.HttpConnectionManager) {
				a.True(notContains(connManager.HttpFilters, wellknown.HealthCheck))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.option.build()
			a := assert.New(t)
			a.Nil(err)
			tc.assertFunc(a, actual)
			a.Equal(wellknown.Router, actual.HttpFilters[len(actual.HttpFilters)-1].Name) // Router must be last
		})
	}
}
