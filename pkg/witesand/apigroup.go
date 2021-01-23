package witesand

import (
	"encoding/json"
	"fmt"
	_ "io/ioutil"
	"net/http"

	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/service"
)

func NewWitesandCatalog(kubeClient kubernetes.Interface, clusterId string, masterOsmIP string) *WitesandCatalog {
	wc := WitesandCatalog{
		myIP:        "",
		masterOsmIP: masterOsmIP,
		clusterId:   clusterId,
		remoteK8s:   make(map[string]RemoteK8s),
		kubeClient:  kubeClient,
	}

	if masterOsmIP != "" {
		wc.UpdateRemoteK8s(masterOsmIP, "master")
	}

	return &wc
}

func (ws *WitesandCatalog) RegisterMyIP(ip string) {
	ws.myIP = ip
}

func (ws *WitesandCatalog) GetMyIP() string {
	return ws.myIP
}

// update the context with received remotePods
func (ws *WitesandCatalog) UpdateRemoteK8s(remoteIP string, remoteClusterId string) {
	remoteK8, exists := ws.remoteK8s[remoteClusterId]
	if exists {
		if remoteK8.OsmIP != remoteIP {
			// IP address changed ?
			remoteK8.OsmIP = remoteIP
		}
	} else {
		remoteK8 = RemoteK8s{
			OsmIP: remoteIP,
		}
		ws.remoteK8s[remoteClusterId] = remoteK8
	}
}

func (ws *WitesandCatalog) UpdateApigroupMap(w http.ResponseWriter, method string, r *http.Request) {

	var input ApigroupToPodMap
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil  {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Decode error! please check your JSON formating.")
		return
	}

	if method == "POST" {
		_, exists := ws.apigroupToPodMap[input.Apigroup]
		if exists {
			w.WriteHeader(400)
			fmt.Fprintf(w, "apigroup %s already exists.", input.Apigroup)
			return
		}
		ws.apigroupToPodMap[input.Apigroup] = input
		// ws.UpdatePodToApigroupMap(nil, input)
		return
	}
	if method == "PUT" {
		o, exists := ws.apigroupToPodMap[input.Apigroup]
		if !exists {
			w.WriteHeader(400)
			fmt.Fprintf(w, "apigroup %s doest not exist.", input.Apigroup)
			return
		}
		if input.Revision < o.Revision {
			w.WriteHeader(400)
			fmt.Fprintf(w, "apigroup %s, revision(old:%d new:%d) stale.", input.Apigroup, o.Revision, input.Revision)
			return
		}
		ws.apigroupToPodMap[input.Apigroup] = input
		//ws.UpdatePodToApigroupMap(o, input)
		return
	}
	if method == "DELETE" {
		_, exists := ws.apigroupToPodMap[input.Apigroup]
		if !exists {
			w.WriteHeader(400)
			fmt.Fprintf(w, "apigroup %s doest not exist.", input.Apigroup)
			return
		}
		delete(ws.apigroupToPodMap, input.Apigroup)
		//ws.UpdatePodToApigroupMap(o, input)
	}
}

func (ws *WitesandCatalog) UpdatePodToApigroupMap(older, newer *ApigroupToPodMap) {
	if older != nil {
		for _, pod := range older.Pods {
			podGroup, exists := ws.podToApigroupMap[pod]
			if !exists {
				continue
			}
			var newApigroups []string
			for _, apigroup := range podGroup.apigroups {
				if apigroup == older.Apigroup {
					continue
				}
				newApigroups = append(newApigroups, apigroup)
			}
			podGroup.apigroups = newApigroups
		}
	}
	for _, pod := range newer.Pods {
		podGroup, exists := ws.podToApigroupMap[pod]
		if !exists {
			podGroup = PodToApigroupMap{
				pod: pod,
			}
		}
		// TODO podGroup.apigroups = append(podGroup.apigroups, apigroup)
		ws.podToApigroupMap[pod] = podGroup
	}
}

func (ws *WitesandCatalog) ListApigroupClusterNames() ([]string, error) {
	var strs []string
	// TODO
	return strs, nil
}

func (ws *WitesandCatalog) ListApigroupToPodIPs() ([]ApigroupToPodIPMap, error) {
	var atopipMap []ApigroupToPodIPMap
	// TODO
	return atopipMap, nil
}

func (ws *WitesandCatalog) IsWSGatewayService(svc service.MeshServicePort) bool {
	// TODO
	return true
}

func (ws *WitesandCatalog) ListRemoteK8s() map[string]RemoteK8s {
	// TODO LOCK
	remoteK8s := ws.remoteK8s

	return remoteK8s
}
