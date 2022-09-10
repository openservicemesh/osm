package osm

import (
	"context"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/workerpool"
)

const (
	workerPoolSize = 0
)

// ProxyUpdater is an abstraction over a type that updates a proxy with a Config of type `T` to the proxy passed in
// the UpdateProxy method.
type ProxyUpdater[T any] interface {
	UpdateProxy(context.Context, *envoy.Proxy, T) error
}

// ProxyConfigGenerator is an abstraction over a type that generates a Config of type `T` for the proxy passed in the
// GenerateConfig method. It is not meant to actually update the proxy itself.
type ProxyConfigGenerator[T any] interface {
	GenerateConfig(context.Context, *envoy.Proxy) (T, error)
}

// ControlPlane is the central part of OSM, that ties in config generation, proxy updates, the message broker, and
// throttling via the workerpool.
type ControlPlane[T any] struct {
	configServer    ProxyUpdater[T]
	configGenerator ProxyConfigGenerator[T]

	catalog       catalog.MeshCataloger
	proxyRegistry *registry.ProxyRegistry
	certManager   *certificate.Manager
	workqueues    *workerpool.WorkerPool
	msgBroker     *messaging.Broker
}

// NewControlPlane creates a new instance of ControlPlane with the given config type T.
func NewControlPlane[T any](server ProxyUpdater[T],
	generator ProxyConfigGenerator[T],
	catalog catalog.MeshCataloger,
	proxyRegistry *registry.ProxyRegistry,
	certManager *certificate.Manager,
	msgBroker *messaging.Broker,
) *ControlPlane[T] {
	return &ControlPlane[T]{
		configServer:    server,
		configGenerator: generator,
		catalog:         catalog,
		proxyRegistry:   proxyRegistry,
		certManager:     certManager,
		workqueues:      workerpool.NewWorkerPool(workerPoolSize),
		msgBroker:       msgBroker,
	}
}
