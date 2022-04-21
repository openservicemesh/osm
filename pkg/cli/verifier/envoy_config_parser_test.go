package verifier

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

func configFromFileOrFail(t *testing.T, filename string) *Config {
	t.Helper()
	sampleConfig, err := os.ReadFile(filename) //#nosec G304: file inclusion via variable
	if err != nil {
		t.Fatalf("Error opening %s: %v", filename, err)
	}
	cfg, err := parseEnvoyConfig(sampleConfig)
	if err != nil {
		t.Fatal("Error parsing Envoy config dump:", err)
	}
	if cfg == nil {
		t.Fatal("Parsed Envoy config is empty")
	}
	return cfg
}

func TestEnvoyConfigParserBookbuyer(t *testing.T) {
	cfg := configFromFileOrFail(t, "testdata/sample-envoy-config-dump-bookbuyer.json")

	a := assert.New(t)
	// Bootstrap
	{
		a.Equal(cfg.Boostrap.Bootstrap.Node.Id, "38cf2479-bfea-4c1e-a961-f8f8e2b2e8cb.sidecar.bookbuyer.bookbuyer.cluster.local")
	}

	{
		// Clusters
		a.Len(cfg.Clusters.DynamicActiveClusters, 1)
		var actual xds_cluster.Cluster
		err := cfg.Clusters.DynamicActiveClusters[0].Cluster.UnmarshalTo(&actual)
		a.Nil(err)
		a.Equal(actual.Name, "bookstore/bookstore")
	}

	{
		// Listeners
		a.Len(cfg.Listeners.DynamicListeners, 1)
		actual := cfg.Listeners.DynamicListeners[0]
		a.Equal(actual.Name, "outbound-listener")
	}

	{
		// Routes
		a.Len(cfg.Routes.DynamicRouteConfigs, 1)
		var actual xds_route.RouteConfiguration
		err := cfg.Routes.DynamicRouteConfigs[0].RouteConfig.UnmarshalTo(&actual)
		a.Nil(err)
		a.Equal(actual.Name, "rds-outbound")
	}
}

func TestEnvoyConfigParserBookstore(t *testing.T) {
	cfg := configFromFileOrFail(t, "testdata/sample-envoy-config-dump-bookstore.json")

	a := assert.New(t)
	// Bootstrap
	{
		a.Equal("b2d941c7-484a-4cd4-ad65-76e41b79e48a.sidecar.bookstore-v1.bookstore.cluster.local", cfg.Boostrap.Bootstrap.Node.Id)
	}

	{
		// Clusters
		a.Len(cfg.Clusters.DynamicActiveClusters, 3)
		var actual xds_cluster.Cluster
		err := cfg.Clusters.DynamicActiveClusters[0].Cluster.UnmarshalTo(&actual)
		a.Nil(err)
		a.Equal("bookstore/bookstore-v1-local", actual.Name)
	}

	{
		// Listeners
		a.Len(cfg.Listeners.DynamicListeners, 3)
		actual := cfg.Listeners.DynamicListeners[0]
		a.Equal("outbound-listener", actual.Name)
	}

	{
		// Routes
		a.Len(cfg.Routes.DynamicRouteConfigs, 2)
		var actual xds_route.RouteConfiguration
		err := cfg.Routes.DynamicRouteConfigs[0].RouteConfig.UnmarshalTo(&actual)
		a.Nil(err)
		a.Equal("rds-outbound", actual.Name)
	}
}
