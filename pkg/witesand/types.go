package witesand

import(
	"net/http"

	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("witesand")
)

const(
	GatewayServiceName = "gateway"
	HttpServerPort = "2500"
	HttpRemoteAddrHeader = "X-Osm-Origin-Ip"
	HttpRemoteClusterIdHeader = "X-Osm-Cluster-Id"
)

type WitesandCatalog struct {
	myIP        string
	masterOsmIP string
	remoteK8s   map[string]RemoteK8s

	apigroupToPodMap   map[string]ApigroupToPodMap
	apigroupToPodIPMap map[string]ApigroupToPodIPMap
	podToApigroupMap   map[string]PodToApigroupMap
}

type ApigroupToPodMap struct {
	Apigroup string      `json:"apigroup"`
	Pods     []string    `json:"pods"`
	Revision int         `json:"revision"`
}

type RemoteK8s struct {
	OsmIP	  string
	failCount int // how many times response not received
}

type RemotePods struct {
	PodToIPs map[string]string
}

type ApigroupToPodIPMap struct {
	Apigroup string
	PodIPs   []string
}

type PodToApigroupMap struct {
	pod       string
	apigroups []string
}

type WitesandCataloger interface {
	RegisterMyIP(ip string)
	GetMyIP() string

	UpdateRemoteK8s(remoteIP string, remoteClusterId string)
	ListRemoteK8s() map[string]RemoteK8s

	UpdateApigroupMap(w http.ResponseWriter, method string, r *http.Request)

	ListApigroupClusterNames() ([]string, error)

	ListApigroupToPodIPs() ([]ApigroupToPodIPMap, error)


	IsWSGatewayService(svc service.MeshServicePort) bool
}

