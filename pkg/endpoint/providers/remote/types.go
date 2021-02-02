package remote

import (
	"github.com/openservicemesh/osm/pkg/smi"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/witesand"
)

var (
	log = logger.New("remote-provider")
)

type ServiceToEndpointMap struct {
	endpoints map[string][]endpoint.Endpoint // key=serviceName
}

// CacheCollection is a struct of the remote services used in OSM
type CacheCollection struct {
	k8sToServiceEndpoints map[string]*ServiceToEndpointMap // key=k8s
}

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	caches         *CacheCollection
	wsCatalog      *witesand.WitesandCatalog
	providerIdent  string
	clusterId      string
	meshSpec       smi.MeshSpec
	announcements  chan announcements.Announcement
}
