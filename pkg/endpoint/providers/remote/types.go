package remote

import (
	"github.com/openservicemesh/osm/pkg/smi"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/namespace"
)

var (
	log = logger.New("remote-provider")
)

// CacheCollection is a struct of the remote services used in OSM
type CacheCollection struct {
	endpoints	map[string][]endpoint.Endpoint
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches              *CacheCollection
	providerIdent       string
	clusterId           string
	meshSpec            smi.MeshSpec
	remoteOsm           string
	announcements       chan interface{}
	namespaceController namespace.Controller
}
