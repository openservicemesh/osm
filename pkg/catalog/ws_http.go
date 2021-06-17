package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/openservicemesh/osm/pkg/witesand"
)

var (
	InitialSyncingPeriod = 3
	QueryTimeout = 1 * time.Minute
	QueryErr = errors.New("rest query timed out")
)

func (mc *MeshCatalog) witesandHttpServerAndClient() {
	go mc.witesandHttpServer()
	go mc.witesandHttpClient()
}

func (mc *MeshCatalog) witesandHttpServer() {
	// GET local edgepods, also learn remote OSM clusterID and IP
	http.HandleFunc("/localedgepods", mc.GetLocalEdgePods) // inter OSM
	http.HandleFunc("/localallpods", mc.GetAllLocalPods)         // inter OSM

	// GET handlers
	http.HandleFunc("/alledgepods", mc.GetAllEdgePods) // from waves
	http.HandleFunc("/edgepod", mc.GetAllEdgePods)     // from waves, will deprecate
	http.HandleFunc("/endpoints", mc.GetLocalEndpoints)      // inter OSM

	http.HandleFunc("/allpods", mc.GetAllPods) // from waves

	// POST/PUT/DELETE handler
	http.HandleFunc("/apigroupMap", mc.ApigroupMapping) // from waves

	http.ListenAndServe(":"+witesand.HttpServerPort, nil)
}

// HTTP client to query other OSMs for pods
func (mc *MeshCatalog) witesandHttpClient() {
	wc := mc.GetWitesandCataloger()

	initialWavesSyncDone := false
	ticker := time.NewTicker(15 * time.Second)

	// run forever
	for {
		// learn local edgepods
		localPods, err := wc.ListLocalEdgePods()
		if err == nil {
			wc.UpdateClusterPods(witesand.LocalClusterId, localPods)
		}
		// learn remote edgepods
		for clusterId, remoteK8s := range wc.ListRemoteK8s() {
			remoteEdgePods, err := mc.QueryRemoteEdgePods(wc, remoteK8s.OsmIP)
			if err == nil {
				wc.UpdateClusterPods(clusterId, &remoteEdgePods)
				wc.UpdateRemoteFailCount(clusterId)
			} else {
				log.Error().Msgf(" witesandHttpClient remove remotek8s clusterID=%d, err=%s", clusterId, err)
				// not responding, trigger remove
				wc.UpdateRemoteK8s(clusterId, "")
			}
		}

		// learn all local pods
		localPods, err = wc.ListAllLocalPods()
		if err == nil {
			wc.UpdateAllPods(witesand.LocalClusterId, localPods)
		}
		// learn all remote pods
		for clusterId, remoteK8s := range wc.ListRemoteK8s() {
			allRemotePods, err := mc.QueryAllPodRemote(wc, remoteK8s.OsmIP)
			if err == nil {
				wc.UpdateAllPods(clusterId, &allRemotePods)
				wc.UpdateRemoteFailCount(clusterId)
			} else {
				log.Error().Msgf(" witesandHttpClient remove remotek8s clusterID=%s err=%s", clusterId, err)
				// not responding, trigger remove
				wc.UpdateRemoteK8s(clusterId, "")
			}
		}

		// learn apigroups from waves
		if wc.IsMaster() && !initialWavesSyncDone && InitialSyncingPeriod != 0 {
			wavesPods, err := wc.ListWavesPodIPs()
			if err == nil && len(wavesPods) != 0 {
				apigroupMaps, err := mc.QueryWaves(wavesPods[0])
				if err == nil {
					wc.UpdateAllApigroupMaps(apigroupMaps)
					initialWavesSyncDone = true
				} else if err == QueryErr {
					log.Error().Msgf(" witesandHttpClient QueryWaves timedout")
				}
			}
		}

		if InitialSyncingPeriod != 0 {
			InitialSyncingPeriod -= 1
		}
		<-ticker.C
	}
}

func (mc *MeshCatalog) QueryWaves(wavesIP string) (*map[string][]string, error) {
	closeChan := make(chan bool)
	var apigroupToPodMaps map[string][]string
	var err error
	go func(wavesIP string) {
		defer close(closeChan)

		log.Info().Msgf("[queryWaves] querying waves:%s", wavesIP)
		dest := fmt.Sprintf("%s:%s", wavesIP, witesand.WavesServerPort)
		url := fmt.Sprintf("http://%s/apigrpgwmap", dest)
		client := &http.Client{}
		req, _ := http.NewRequest("GET", url, nil)
		var resp *http.Response
		resp, err = client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			var b []byte
			b, err = ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(b, &apigroupToPodMaps)
				if err == nil {
					log.Info().Msgf("[queryWaves] wavesIP:%s apigroupToPodMaps:%+v", wavesIP, apigroupToPodMaps)
					return
				} else {
					log.Error().Msgf("[queryWaves] Marshalling error:%s body:%+v", err, b)
					return
				}
			}
		}
		log.Info().Msgf("[queryWaves] err:%+v", err)
		return
	}(wavesIP)

	select {
	case <-time.After(QueryTimeout):
		log.Error().Msgf("[QueryRemoteEdgePods] failed. Timeout")
		return &apigroupToPodMaps, QueryErr
	case <-closeChan:
	}
	return &apigroupToPodMaps, err
}

func (mc *MeshCatalog) QueryAllPodRemote(wc witesand.WitesandCataloger, remoteOsmIP string) (witesand.ClusterPods, error) {
	closeChan := make(chan bool)
	var err error
	var remotePods witesand.ClusterPods
	go func(remoteOsmIP string){
		defer close(closeChan)

		log.Info().Msgf("[queryAllPodRemote] querying osm:%s", remoteOsmIP)
		dest := fmt.Sprintf("%s:%s", remoteOsmIP, witesand.HttpServerPort)
		url := fmt.Sprintf("http://%s/localallpods", dest)
		client := &http.Client{}
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set(witesand.HttpRemoteAddrHeader, mc.getMyIP(remoteOsmIP))
		req.Header.Set(witesand.HttpRemoteClusterIdHeader, wc.GetClusterId())
		var resp *http.Response
		resp, err = client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			var b []byte
			b, err = ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(b, &remotePods)
				if err == nil {
					log.Info().Msgf("[queryAllPodRemote] remoteOsmIP:%s remotePods:%+v", remoteOsmIP, remotePods)
					return
				} else {
					log.Error().Msgf("[queryAllPodRemote] Marshalling error:%s", err)
					return
				}
			}
		}
		log.Info().Msgf("[queryAllPodRemote] err:%+v", err)
	} (remoteOsmIP)

	select {
	case <-time.After(QueryTimeout):
		log.Error().Msgf("[QueryRemoteEdgePods] failed. Timeout")
		return remotePods, QueryErr
	case <-closeChan:
	}
	return remotePods, err
}

func (mc *MeshCatalog) QueryRemoteEdgePods(wc witesand.WitesandCataloger, remoteOsmIP string) (witesand.ClusterPods, error) {
	var err error
	var remotePods witesand.ClusterPods
	closeChan := make(chan bool)
	go func(remoteOsmIP string) {
		defer close(closeChan)

		log.Info().Msgf(" witesandHttpClient [QueryRemoteEdgePods] querying osm:%s", remoteOsmIP)
		dest := fmt.Sprintf("%s:%s", remoteOsmIP, witesand.HttpServerPort)
		url := fmt.Sprintf("http://%s/localedgepods", dest)
		client := &http.Client{}
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set(witesand.HttpRemoteAddrHeader, mc.getMyIP(remoteOsmIP))
		req.Header.Set(witesand.HttpRemoteClusterIdHeader, wc.GetClusterId())
		var resp *http.Response
		resp, err = client.Do(req)
		//var remotePods witesand.ClusterPods
		if err == nil {
			defer resp.Body.Close()
			var b []byte
			b, err = ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(b, &remotePods)
				if err == nil {
					log.Info().Msgf(" witesandHttpClient [QueryRemoteEdgePods] remoteOsmIP:%s remotePods:%+v", remoteOsmIP, remotePods)
					return
				} else {
					log.Error().Msgf("witesandHttpClient [QueryRemoteEdgePods] Marshalling error:%s", err)
					return
				}
			}
		}
		log.Info().Msgf("witesandHttpClient [QueryRemoteEdgePods] err:%+v", err)
		return
	}(remoteOsmIP)

	select {
	case <-time.After(QueryTimeout):
		log.Error().Msgf("[QueryRemoteEdgePods] failed. Timeout")
		return remotePods, QueryErr
	case <-closeChan:
	}
	return remotePods, err
}

func (mc *MeshCatalog) getMyIP(destIP string) string {
	// Get preferred outbound ip of this machine
	myIP := mc.GetWitesandCataloger().GetMyIP()
	if myIP != "" {
		return myIP
	}
	dest := fmt.Sprintf("%s:%s", destIP, witesand.HttpServerPort)
	conn, err := net.Dial("udp", dest)
	if err != nil {
		log.Error().Msgf("[getMyIP] err:%s", err)
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	myIP = localAddr.IP.String()

	// cache myIP for future use
	mc.GetWitesandCataloger().RegisterMyIP(myIP)

	return myIP
}

func (mc *MeshCatalog) GetLocalEdgePods(w http.ResponseWriter, r *http.Request) {
	// learn remote OSM clusterID and address
	remoteAddress := r.Header.Get(witesand.HttpRemoteAddrHeader)
	remoteClusterId := r.Header.Get(witesand.HttpRemoteClusterIdHeader)

	//log.Info().Msgf("[GetLocalEdgePods] remote IP:%s clusterId:%s", remoteAddress, remoteClusterId)
	mc.GetWitesandCataloger().UpdateRemoteK8s(remoteClusterId, remoteAddress)

	list, err := mc.GetWitesandCataloger().ListLocalEdgePods()
	if err != nil {
		log.Error().Msgf("err fetching local edgepod %+v", err)
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Error().Msgf("err fetching local edgepod %+v", err)
	}
}

func (mc *MeshCatalog) GetAllLocalPods(w http.ResponseWriter, r *http.Request) {
	// learn remote OSM clusterID and address
	remoteAddress := r.Header.Get(witesand.HttpRemoteAddrHeader)
	remoteClusterId := r.Header.Get(witesand.HttpRemoteClusterIdHeader)

	//log.Info().Msgf("[GetAllLocalPods] remote IP:%s clusterId:%s", remoteAddress, remoteClusterId)
	mc.GetWitesandCataloger().UpdateRemoteK8s(remoteClusterId, remoteAddress)

	list, err := mc.GetWitesandCataloger().ListAllLocalPods()
	if err != nil {
		log.Error().Msgf("err fetching local edgepod %+v", err)
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Error().Msgf("err fetching local edgepod %+v", err)
	}
}

func (mc *MeshCatalog) GetAllEdgePods(w http.ResponseWriter, r *http.Request) {
	if InitialSyncingPeriod != 0 {
		// initial cooling period, need to wait till we sync with others
		log.Error().Msgf("InitialSyncingPeriod not over !!, send error response")
		w.WriteHeader(503)
		fmt.Fprintf(w, "Not ready")
		return
	}
	list, err := mc.GetWitesandCataloger().ListAllEdgePods()
	if err != nil {
		log.Error().Msgf("err fetching edgepod %+v", err)
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Error().Msgf("err fetching edgepod %+v", err)
	}
}

func (mc *MeshCatalog) GetAllPods(w http.ResponseWriter, r *http.Request) {
	if InitialSyncingPeriod != 0 {
		// initial cooling period, need to wait till we sync with others
		log.Error().Msgf("InitialSyncingPeriod not over !!, send error response")
		w.WriteHeader(503)
		fmt.Fprintf(w, "Not ready")
		return
	}
	list, err := mc.GetWitesandCataloger().ListAllPods()
	if err != nil {
		log.Error().Msgf("err fetching edgepod %+v", err)
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Error().Msgf("err fetching edgepod %+v", err)
	}
}

func (mc *MeshCatalog) GetLocalEndpoints(w http.ResponseWriter, r *http.Request) {
	log.Info().Msgf("[GetLocalEndpoints] invoked")
	endpointMap, err := mc.ListLocalClusterEndpoints()
	if err != nil {
		log.Error().Msgf("err fetching endpoints %+v", err)
	}

	if err := json.NewEncoder(w).Encode(endpointMap); err != nil {
		log.Error().Msgf("err encoding endpoints %+v", err)
	}
}

func (mc *MeshCatalog) ApigroupMapping(w http.ResponseWriter, r *http.Request) {
	mc.witesandCatalog.UpdateApigroupMap(w, r)
}
