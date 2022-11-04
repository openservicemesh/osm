package generator

import (
	"context"
	"sync"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/models"
)

var (
	log = logger.New("envoy/generator")
)

// EnvoyConfigGenerator is used to generate all xDS response types per proxy.
type EnvoyConfigGenerator struct {
	catalog        *catalog.MeshCatalog
	generators     map[envoy.TypeURI]func(context.Context, *models.Proxy) ([]types.Resource, error)
	certManager    *certificate.Manager
	xdsMapLogMutex sync.Mutex
	xdsLog         map[string]map[envoy.TypeURI][]time.Time
}

// NewEnvoyConfigGenerator creates a new instance of EnvoyConfigGenerator.
func NewEnvoyConfigGenerator(catalog *catalog.MeshCatalog, certManager *certificate.Manager) *EnvoyConfigGenerator {
	g := &EnvoyConfigGenerator{
		catalog:     catalog,
		certManager: certManager,
		xdsLog:      make(map[string]map[envoy.TypeURI][]time.Time),
	}
	g.generators = map[envoy.TypeURI]func(context.Context, *models.Proxy) ([]types.Resource, error){
		envoy.TypeCDS: g.generateCDS,
		envoy.TypeEDS: g.generateEDS,
		envoy.TypeLDS: g.generateLDS,
		envoy.TypeRDS: g.generateRDS,
		envoy.TypeSDS: g.generateSDS,
	}
	return g
}

// GenerateConfig generates and returns the resources for the given proxy.
func (g *EnvoyConfigGenerator) GenerateConfig(ctx context.Context, proxy *models.Proxy) (map[string][]types.Resource, error) {
	cacheResourceMap := map[string][]types.Resource{}
	for typeURI, handler := range g.generators {
		log.Trace().Str("proxy", proxy.String()).Msgf("Getting resources for type %s", typeURI.Short())

		if g.catalog.GetMeshConfig().Spec.Observability.EnableDebugServer {
			g.trackXDSLog(proxy.UUID.String(), typeURI)
		}

		startedAt := time.Now()
		resources, err := handler(ctx, proxy)
		xdsPathTimeTrack(startedAt, typeURI, proxy, err == nil)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGeneratingReqResource)).Str("proxy", proxy.String()).
				Msgf("Error generating response for typeURI: %s", typeURI.Short())
			xdsPathTimeTrack(time.Now(), envoy.TypeADS, proxy, false)
			return nil, err
		}

		cacheResourceMap[typeURI.String()] = resources
	}

	xdsPathTimeTrack(time.Now(), envoy.TypeADS, proxy, true)
	return cacheResourceMap, nil
}
