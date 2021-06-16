package witesand

import (
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
	"github.com/openservicemesh/osm/pkg/service"
)

func NewWitesandCatalog(kubeClient kubernetes.Interface, clusterId string) *WitesandCatalog {
	wc := WitesandCatalog{
		myIP:               "",
		masterOsmIP:        "",
		clusterId:          clusterId,
		remoteK8s:          make(map[string]RemoteK8s),
		clusterPodMap:      make(map[string]ClusterPods),
		allPodMap:          make(map[string]ClusterPods),
		kubeClient:         kubeClient,
		apigroupToPodMap:   make(map[string]ApigroupToPodMap),
		apigroupToPodIPMap: make(map[string]ApigroupToPodIPMap),
	}

	wc.UpdateMasterOsmIP()
	wc.UpdateUnicastSvcs()

	return &wc
}

// cache myIP
func (wc *WitesandCatalog) RegisterMyIP(ip string) {
	log.Info().Msgf("[RegisterMyIP] myIP:%s", ip)
	wc.myIP = ip
}

func (wc *WitesandCatalog) GetMyIP() string {
	return wc.myIP
}

// read env to update masterOsmIP (in case master OSM restarted)
func (wc *WitesandCatalog) UpdateMasterOsmIP() {
	newIP := os.Getenv("MASTER_OSM_IP")
	if newIP != "" {
		log.Info().Msgf("[RegisterMasterOsmIP] masterOsmIP:%s", newIP)
		wc.UpdateRemoteK8s("master", newIP)
		wc.masterOsmIP = newIP
	}
}

func (wc *WitesandCatalog) UpdateUnicastSvcs() {
	wc.unicastEnabledSvcs = make([]string, 0)
	wc.unicastEnabledSvcs = append(wc.unicastEnabledSvcs, "edgepod") // by default add gw
	unicastSvcsString := os.Getenv("UNICAST_ENABLED_SERVICES")
	if unicastSvcsString != "" {
		wc.unicastEnabledSvcs = append(wc.unicastEnabledSvcs, strings.Split(unicastSvcsString, ",")...)
		log.Info().Msgf("[UpdateUnicastSvcs] unicastEnabledSvcs:%+v", wc.unicastEnabledSvcs)
	}
}

func (wc *WitesandCatalog) IsMaster() bool {
	return wc.masterOsmIP == ""
}

func (wc *WitesandCatalog) UpdateRemoteFailCount(clusterId string) {
	remoteK8, exists := wc.remoteK8s[clusterId]
	if exists {
		remoteK8.failCount = 0
	}
}

// update the context with received remoteK8s
func (wc *WitesandCatalog) UpdateRemoteK8s(remoteClusterId string, remoteIP string) {
	if remoteClusterId == "" {
		log.Error().Msgf("[UpdateRemoteK8s] clusterId:%s remoteIP=%s", remoteClusterId, remoteIP)
		return
	}

	// handle the case of remoteIP not responding, remove it from the list after certain retries
	if remoteIP == "" {
		remoteK8, exists := wc.remoteK8s[remoteClusterId]
		if exists {
			remoteK8.failCount += 1
			if remoteK8.failCount >= 3 {
				log.Info().Msgf("[UpdateRemoteK8s] Delete clusterId:%s", remoteClusterId)
				delete(wc.remoteK8s, remoteClusterId)
				wc.UpdateClusterPods(remoteClusterId, nil)
				wc.UpdateAllPods(remoteClusterId, nil)
				return
			}
			wc.remoteK8s[remoteClusterId] = remoteK8
		}
		return
	}

	log.Info().Msgf("[UpdateRemoteK8s] IP:%s clusterId:%s", remoteIP, remoteClusterId)
	remoteK8, exists := wc.remoteK8s[remoteClusterId]
	if exists {
		if remoteK8.OsmIP != remoteIP {
			log.Info().Msgf("[UpdateRemoteK8s] update IP:%s clusterId:%s", remoteIP, remoteClusterId)
			// IP address changed ?
			wc.remoteK8s[remoteClusterId] = RemoteK8s{
				OsmIP: remoteIP,
			}
		}
	} else {
		log.Info().Msgf("[UpdateRemoteK8s] create IP:%s clusterId:%s", remoteIP, remoteClusterId)
		wc.remoteK8s[remoteClusterId] = RemoteK8s{
			OsmIP: remoteIP,
		}
	}
}

func (wc *WitesandCatalog) ListRemoteK8s() map[string]RemoteK8s {
	// TODO LOCK
	remoteK8s := wc.remoteK8s

	return remoteK8s
}

func (wc *WitesandCatalog) IsWSEdgePodService(svc service.MeshServicePort) bool {
	return strings.HasPrefix(svc.Name, "edgepod")
}

func (wc *WitesandCatalog) IsWSUnicastService(inputSvcName string) bool {
	for _, prefix := range wc.unicastEnabledSvcs {
		if prefix == "" {
			// not needed, but safer
			continue
		}
		if strings.HasPrefix(inputSvcName, prefix) {
			return true
		}
	}
	return false
}

func (wc *WitesandCatalog) updateEnvoy() {
	events.GetPubSubInstance().Publish(events.PubSubMessage{
		AnnouncementType: announcements.ScheduleProxyBroadcast,
		NewObj:           nil,
		OldObj:           nil,
	})
}
