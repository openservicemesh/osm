package ads

import (
	"context"
	"sync"

	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/utils"
)

var errProxyNotFound = errors.New("proxy not found")

var snapshotCache cache.SnapshotCache

type callbacks struct {
	xdsCertificateCommonName certificate.CommonName
	proxyFromStream          sync.Map
	catalog                  catalog.MeshCataloger
	proxyRegistry            *registry.ProxyRegistry
}

// NewestServer creates a v3 of go-control-plane
func (s *Server) NewestServer() xds.Server {
	snapshotCache = cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	cb := &callbacks{
		catalog:       s.catalog,
		proxyRegistry: s.proxyRegistry,
	}
	server := xds.NewServer(context.Background(), snapshotCache, cb)
	return server
}

// OnStreamOpen is called once an xDS stream is open with a stream ID and the type URL (or "" for ADS).
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (cb *callbacks) OnStreamOpen(ctx context.Context, streamID int64, typeURL string) error {
	log.Info().Msgf("Envoy connected to streamID=%d for typeURL=%s", streamID, typeURL)
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	certCommonName, certSerialNumber, err := utils.ValidateClient(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "Could not start Aggregated Discovery Service gRPC stream for newly connected Envoy proxy")
	}
	cb.xdsCertificateCommonName = certCommonName

	log.Info().Msgf("Envoy connected to with CN=%s and SerialNumber=%s", certCommonName, certSerialNumber)

	// TODO(draychev): check for envoy.ErrTooManyConnections; GitHub Issue https://github.com/openservicemesh/osm/issues/2332

	log.Trace().Msgf("Envoy with certificate SerialNumber=%s connected", certSerialNumber)
	metricsstore.DefaultMetricsStore.ProxyConnectCount.Inc()

	// This is the Envoy proxy that just connected to the control plane.
	// NOTE: This is step 1 of the registration. At this point we do not yet have context on the Pod.
	//       Details on which Pod this Envoy is fronting will arrive via xDS in the NODE_ID string.
	//       When this arrives we will call RegisterProxy() a second time - this time with Pod context!
	proxy := envoy.NewProxy(certCommonName, certSerialNumber, utils.GetIPFromContext(ctx))

	cb.proxyFromStream.Store(streamID, proxy)

	return nil
}

// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
func (cb *callbacks) OnStreamClosed(streamID int64) {
	log.Info().Msgf("Envoy streamID=%d closed", streamID)
}

// OnStreamRequest is called once a request is received on a stream.
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (cb *callbacks) OnStreamRequest(streamID int64, req *envoy_service_discovery_v3.DiscoveryRequest) error {
	log.Info().Msgf("Request from Envoy streamID=%d node.ID=%s", streamID, req.Node.GetId())
	proxyInterface, ok := cb.proxyFromStream.Load(streamID)
	if !ok {
		return errProxyNotFound
	}
	if err := recordEnvoyPodMetadata(req, proxyInterface.(*envoy.Proxy), cb.proxyRegistry); err != nil {
		log.Err(err).Msgf("Error recording proxy metadata")
	}
	updateConfig(cb.xdsCertificateCommonName)
	return nil
}

// OnStreamResponse is called immediately prior to sending a response on a stream.
func (cb *callbacks) OnStreamResponse(int64, *envoy_service_discovery_v3.DiscoveryRequest, *envoy_service_discovery_v3.DiscoveryResponse) {

}

// OnFetchRequest is for REST calls and is not implemented.
func (cb *callbacks) OnFetchRequest(context.Context, *envoy_service_discovery_v3.DiscoveryRequest) error {
	panic("NOT IMPLEMENTED")
}

// OnFetchResponse is for REST calls and is not implemented.
func (cb *callbacks) OnFetchResponse(*envoy_service_discovery_v3.DiscoveryRequest, *envoy_service_discovery_v3.DiscoveryResponse) {
	panic("NOT IMPLEMENTED")
}

func updateConfig(cn certificate.CommonName) {
	version := "1234" // string(node.Version)
	var endpoints []types.Resource
	var clusters []types.Resource
	var routes []types.Resource
	var listeners []types.Resource
	var runtimes []types.Resource
	var secrets []types.Resource
	snapshot := cache.NewSnapshot(version, endpoints, clusters, routes, listeners, runtimes, secrets)
	if err := snapshot.Consistent(); err != nil {
		log.Err(err).Msgf("Inconsistent cache")
		// TODO(draychev): return error?
		return
	}
	if err := snapshotCache.SetSnapshot(cn.String(), snapshot); err != nil {
		log.Err(err).Msgf("Error setting snapshot cache")
		// TODO(draychev): return error?
		return
	}
}
