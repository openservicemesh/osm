package ads

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
)

// proxyResponseJob is the workerpool job implementation for a Proxy response function
// It takes the parameters of `server.sendResponse` and allows to queue it as a job on a workerpool
type proxyResponseJob struct {
	typeurls  mapset.Set
	proxy     *envoy.Proxy
	cfg       configurator.Configurator
	adsStream *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer
	request   *xds_discovery.DiscoveryRequest
	xdsServer *Server

	// Optional waiter
	done chan struct{}
}

// Run implementation for `server.sendResponse` job
func (proxyJob *proxyResponseJob) Run() {
	err := (*proxyJob.xdsServer).sendResponse(proxyJob.typeurls, proxyJob.proxy, proxyJob.adsStream, proxyJob.request, proxyJob.cfg)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create and send %v update to Envoy with xDS Certificate SerialNumber=%s for PodUUID=%s",
			proxyJob.typeurls, proxyJob.proxy.GetCertificateSerialNumber(), proxyJob.proxy.GetPodUID())
	}
	close(proxyJob.done)
}

// JobName implementation for this job, for logging purposes
func (proxyJob *proxyResponseJob) JobName() string {
	return fmt.Sprintf("sendJob-%s", proxyJob.proxy.GetCertificateSerialNumber())
}

// Hash implementation for this job to hash into the worker queues
func (proxyJob *proxyResponseJob) Hash() uint64 {
	// Uses proxy hash to always serialize work for the same proxy to the same worker,
	// this avoid out-of-order misshandling of envoy updates by multiple workers
	return proxyJob.proxy.GetHash()
}
