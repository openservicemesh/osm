package witesand

import (
	"net/http"

	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("witesand")
)

const (
	EdgePodServiceName = "default/edgepod"

	// HTTP REST port for communication with WAVES and remote OSMes
	HttpServerPort  = "2500"
	WavesServerPort = "9053"

	// HTTP headers for remote OSMes
	HttpRemoteAddrHeader      = "X-Osm-Origin-Ip"
	HttpRemoteClusterIdHeader = "X-Osm-Cluster-Id"

	// HTTP headers used by envoy to route traffic to edgepod clusters
	WSClusterHeader = "x-ws-dest-cluster"
	WSHashHeader    = "x-ws-hash-header"

	// Cluster name suffixes for ring-hash LB clusters
	DeviceHashSuffix = "-device-hash"

	LocalClusterId = "local"
)

type WitesandCatalog struct {
	myIP        string
	clusterId   string
	masterOsmIP string

	unicastEnabledSvcs []string
	remoteK8s          map[string]RemoteK8s   // key = clusterId
	clusterPodMap      map[string]ClusterPods // key = clusterId
	allPodMap          map[string]ClusterPods // key = clusterId

	kubeClient kubernetes.Interface

	apigroupToPodMap   map[string]ApigroupToPodMap
	apigroupToPodIPMap map[string]ApigroupToPodIPMap
}

type RemoteK8s struct {
	OsmIP     string
	failCount int // how many times response not received
}

type ClusterPods struct {
	PodToIPMap map[string]string `json:"podtoipmap"`
}

type ApigroupToPodMap struct {
	Apigroup string   `json:"apigroup"`
	Pods     []string `json:"pods"`
	Revision int      `json:"revision"`
}

type ApigroupToPodIPMap struct {
	Apigroup string   `json:"apigroup"`
	PodIPs   []string `json:"podips"`
}

type WitesandCataloger interface {
	RegisterMyIP(ip string)
	GetMyIP() string
	GetClusterId() string
	UpdateMasterOsmIP()
	IsMaster() bool

	UpdateRemoteFailCount(remoteClusterId string)
	UpdateRemoteK8s(remoteClusterId string, remoteIP string)
	UpdateClusterPods(remoteClusterId string, remotePods *ClusterPods)
	UpdateAllPods(ClusterId string, Pods *ClusterPods)
	ListRemoteK8s() map[string]RemoteK8s

	UpdateApigroupMap(w http.ResponseWriter, r *http.Request)
	UpdateAllApigroupMaps(*map[string][]string)

	// for usage by CDS
	ListApigroupClusterNames() ([]string, error)

	// for usage by EDS
	ListApigroupToPodIPs() ([]ApigroupToPodIPMap, error)
	ListAllEdgePodIPs() (*ClusterPods, error)

	ListLocalEdgePods() (*ClusterPods, error)
	ListAllLocalPods() (*ClusterPods, error)
	ListAllEdgePods() ([]string, error)
	ListAllPods() ([]string, error)
	ListWavesPodIPs() ([]string, error)

	IsWSEdgePodService(svc service.MeshServicePort) bool
	IsWSUnicastService(svc string) bool
}
