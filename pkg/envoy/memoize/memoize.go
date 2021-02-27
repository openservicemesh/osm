package memoize

import (
	"time"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/patrickmn/go-cache"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("envoy/memoize")

const (
	defaultExpiration = 1 * time.Second
	cleanupInterval   = 10 * time.Minute
)

var memoizationCache = make(map[string]*cache.Cache)

// Memoize memoizes the function given
func Memoize(
	dsType, key string,
	fn func(catalog.MeshCataloger, *envoy.Proxy, *xds_discovery.DiscoveryRequest, configurator.Configurator, certificate.Manager) (*xds_discovery.DiscoveryResponse, error),
	meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, certIssuer certificate.Manager,
) (*xds_discovery.DiscoveryResponse, error) {
	log.Debug().Msgf("Looking for %s Response in cache first", dsType)

	if memoizationCache[dsType] == nil {
		log.Debug().Msgf("Initializing %s memoization cache", dsType)
		memoizationCache[dsType] = cache.New(defaultExpiration, cleanupInterval)
	}

	if resp, found := memoizationCache[dsType].Get(key); found {
		log.Debug().Msgf("Found in %s memoization cache for key=%s", dsType, key)
		return resp.(*xds_discovery.DiscoveryResponse), nil
	}

	result, err := fn(meshCatalog, proxy, request, cfg, certIssuer)
	if err != nil {
		log.Err(err).Msgf("Error looking in %s memoization cache for key=%s", dsType, key)
		return result, err
	}

	log.Debug().Msgf("Saving newly generated result in %s memoization cache for key=%s", dsType, key)
	memoizationCache[dsType].Set(key, result, defaultExpiration)

	return result, nil
}
