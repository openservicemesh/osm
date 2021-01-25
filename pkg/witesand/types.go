package witesand

import(
	"net/http"

	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("witesand")
)

const(
	GatewayServiceName = "default/gateway"
	HttpServerPort = "2500"
	HttpRemoteAddrHeader = "X-Osm-Origin-Ip"
	HttpRemoteClusterIdHeader = "X-Osm-Cluster-Id"
	DeviceHashSuffix = "-device-hash"
	LocalClusterId = "local"
)

type WitesandCatalog struct {
	myIP        string
	clusterId   string
	masterOsmIP string

	remoteK8s     map[string]RemoteK8s     // key = clusterId
	clusterPodMap map[string]ClusterPods    // key = clusterId

	kubeClient kubernetes.Interface

	apigroupToPodMap   map[string]ApigroupToPodMap
	apigroupToPodIPMap map[string]ApigroupToPodIPMap
}

type RemoteK8s struct {
	OsmIP	  string
	failCount int // how many times response not received
}

type ClusterPods struct {
	PodToIPMap map[string]string  `json:"podtoipmap"`
}

type ApigroupToPodMap struct {
	Apigroup string      `json:"apigroup"`
	Pods     []string    `json:"pods"`
	Revision int         `json:"revision"`
}

type ApigroupToPodIPMap struct {
	Apigroup string      `json:"apigroup"`
	PodIPs   []string    `json:"podips"`
}

type WitesandCataloger interface {
	RegisterMyIP(ip string)
	GetMyIP() string
	GetClusterId() string

	UpdateRemoteK8s(remoteIP string, remoteClusterId string)
	UpdateClusterPods(remoteClusterId string, remotePods *ClusterPods)
	ListRemoteK8s() map[string]RemoteK8s

	UpdateApigroupMap(w http.ResponseWriter, method string, r *http.Request)

	// for usage by CDS
	ListApigroupClusterNames() ([]string, error)

	// for usage by EDS
	ListApigroupToPodIPs() ([]ApigroupToPodIPMap, error)
	ListAllGatewayPodIPs() (*ClusterPods, error)

	ListLocalGatewayPods() (*ClusterPods, error)
	ListAllGatewayPods() ([]string, error)

	IsWSGatewayService(svc service.MeshServicePort) bool
}
