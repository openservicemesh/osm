package catalog

import(
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net"
	"time"

	"github.com/openservicemesh/osm/pkg/witesand"
)

var(
	InitialSyncingPeriod = 3
)

func (mc *MeshCatalog) witesandHttpServerAndClient() {
	go mc.witesandHttpServer()
	go mc.witesandHttpClient()
}

func (mc *MeshCatalog) witesandHttpServer() {
	// GET local gatewaypods, also learn remote OSM clusterID and IP
	http.HandleFunc("/localgatewaypods", mc.GetLocalGatewayPods) // inter OSM

	// GET handlers
	http.HandleFunc("/allgatewaypods", mc.GetAllGatewayPods) // from waves
	http.HandleFunc("/gatewaypod", mc.GetAllGatewayPods) // from waves, will deprecate
	http.HandleFunc("/endpoints", mc.GetLocalEndpoints) // inter OSM

	// POST/PUT/DELETE handler
	http.HandleFunc("/apigroupMap", mc.ApigroupMapping) // from waves

	http.ListenAndServe(":" + witesand.HttpServerPort , nil)
}

// HTTP client to query other OSMs for pods
func (mc *MeshCatalog) witesandHttpClient() {
	wc := mc.GetWitesandCataloger()
	queryRemoteOsm := func(remoteOsmIP string) (witesand.ClusterPods, error) {
		log.Info().Msgf("[queryRemoteOsm] querying osm:%s", remoteOsmIP)
		dest := fmt.Sprintf("%s:%s", remoteOsmIP, witesand.HttpServerPort)
		url := fmt.Sprintf("http://%s/localgatewaypods", dest)
		client := &http.Client{}
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set(witesand.HttpRemoteAddrHeader, mc.getMyIP(remoteOsmIP))
		req.Header.Set(witesand.HttpRemoteClusterIdHeader, wc.GetClusterId())
		resp, err := client.Do(req)
		var remotePods witesand.ClusterPods
		if err == nil {
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(b, &remotePods)
				if err == nil {
					log.Info().Msgf("[queryRemoteOsm] remoteOsmIP:%s remotePods:%+v", remoteOsmIP, remotePods)
					return remotePods, nil
				} else {
					log.Error().Msgf("[queryRemoteOsm] Marshalling error:%s", err)
					return remotePods, err
				}
			}
		}
		log.Info().Msgf("[queryRemoteOsm] err:%+v", err)
		return remotePods, err
	}

	queryWaves := func(wavesIP string) (*map[string][]string, error) {
		log.Info().Msgf("[queryWaves] querying waves:%s", wavesIP)
		dest := fmt.Sprintf("%s:%s", wavesIP, witesand.WavesServerPort)
		url := fmt.Sprintf("http://%s/apigrpgwmap", dest)
		client := &http.Client{}
		req, _ := http.NewRequest("GET", url, nil)
		resp, err := client.Do(req)
		var apigroupToPodMaps map[string][]string
		if err == nil {
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(b, &apigroupToPodMaps)
				if err == nil {
					log.Info().Msgf("[queryWaves] wavesIP:%s apigroupToPodMaps:%+v", wavesIP, apigroupToPodMaps)
					return &apigroupToPodMaps, nil
				} else {
					log.Error().Msgf("[queryWaves] Marshalling error:%s body:%+v", err, b)
					return nil, err
				}
			}
		}
		log.Info().Msgf("[queryWaves] err:%+v", err)
		return nil, err
	}

	initialWavesSyncDone := false
	ticker := time.NewTicker(15 * time.Second)
	// run forever
	for {
		// learn local pods
		localPods, err := wc.ListLocalGatewayPods()
		if err == nil {
			wc.UpdateClusterPods(witesand.LocalClusterId, localPods)
		}
		// learn remote pods
		for clusterId, remoteK8s := range wc.ListRemoteK8s() {
			remotePods, err := queryRemoteOsm(remoteK8s.OsmIP)
			if err == nil {
				wc.UpdateClusterPods(clusterId, &remotePods)
			} else {
				// not responding, trigger remove
				wc.UpdateRemoteK8s(clusterId, "")
			}
		}

		// learn apigroups from waves
		if wc.IsMaster() && !initialWavesSyncDone && InitialSyncingPeriod != 0 {
			wavesPods, err := wc.ListWavesPodIPs()
			if err == nil && len(wavesPods) != 0 {
				apigroupMaps, err := queryWaves(wavesPods[0])
				if err == nil {
					wc.UpdateAllApigroupMaps(apigroupMaps)
					initialWavesSyncDone = true
				}
			}
		}

		if InitialSyncingPeriod != 0 {
			InitialSyncingPeriod -= 1
		}
		<-ticker.C
	}
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

func (mc *MeshCatalog) GetLocalGatewayPods(w http.ResponseWriter, r *http.Request) {
	// learn remote OSM clusterID and address
	remoteAddress := r.Header.Get(witesand.HttpRemoteAddrHeader)
	remoteClusterId := r.Header.Get(witesand.HttpRemoteClusterIdHeader)

	log.Info().Msgf("[GetLocalGatewayPods] remote IP:%s clusterId:%s", remoteAddress, remoteClusterId)
	mc.GetWitesandCataloger().UpdateRemoteK8s(remoteClusterId, remoteAddress)

	list, err := mc.GetWitesandCataloger().ListLocalGatewayPods()
	if err != nil {
		log.Error().Msgf("err fetching local gateway pod %+v", err)
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Error().Msgf("err fetching local gateway pod %+v", err)
	}
}

func (mc *MeshCatalog) GetAllGatewayPods(w http.ResponseWriter, r *http.Request) {
	if InitialSyncingPeriod != 0 {
		// initial cooling period, need to wait till we sync with others
		log.Error().Msgf("InitialSyncingPeriod not over !!, send error response")
		w.WriteHeader(503)
		fmt.Fprintf(w, "Not ready")
		return
	}
	list, err := mc.GetWitesandCataloger().ListAllGatewayPods()
	if err != nil {
		log.Error().Msgf("err fetching gateway pod %+v", err)
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Error().Msgf("err fetching gateway pod %+v", err)
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
