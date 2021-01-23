package catalog

import(
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net"
	"strings"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/witesand"
)

func (mc *MeshCatalog) initWitesandHttpServer() {
	go func() {
		// GET local gatewaypods, also learn remote OSM clusterID and IP
		http.HandleFunc("/localgatewaypods", mc.GetLocalGatewayPods) // inter OSM

		// GET handlers
		http.HandleFunc("/allgatewaypods", mc.GetAllGatewayPods) // from waves
		http.HandleFunc("/endpoints", mc.LocalEndpoints) // inter OSM

		// POST handler
		http.HandleFunc("/apigroupMap", mc.ApigroupMapping)

		http.ListenAndServe(":" + witesand.HttpServerPort , nil)
	}()
}

func (mc *MeshCatalog) GetMyIP() string {
        // Get preferred outbound ip of this machine
	myIP := mc.GetWitesandCataloger().GetMyIP()
	if myIP != "" {
		return myIP
	}
	conn, err := net.Dial("udp", witesand.HttpServerPort)
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	myIP = localAddr.IP.String()

	mc.GetWitesandCataloger().RegisterMyIP(myIP)

	return myIP
}

func (mc *MeshCatalog) GetLocalGatewayPods(w http.ResponseWriter, r *http.Request) {
	// learn remote OSM clusterID and address
	remoteAddress := r.Header.Get(witesand.HttpRemoteAddrHeader)
	remoteClusterId := r.Header.Get(witesand.HttpRemoteClusterIdHeader)

	log.Info().Msgf("[GetLocalGatewayPods] remote IP:%s clusterId:%s", remoteAddress, remoteClusterId)

	mc.GetWitesandCataloger().UpdateRemoteK8s(remoteAddress, remoteClusterId)

	list, err := mc.ListLocalGatewaypods(witesand.GatewayServiceName)
	if err != nil {
		log.Error().Msgf("err fetching local gateway pod %+v", err)
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Error().Msgf("err fetching local gateway pod %+v", err)
	}

}

func (mc *MeshCatalog) GetAllGatewayPods(w http.ResponseWriter, r *http.Request) {
	list, err := mc.ListAllGatewaypods(witesand.GatewayServiceName)
	if err != nil {
		log.Error().Msgf("err fetching gateway pod %+v", err)
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Error().Msgf("err fetching gateway pod %+v", err)
	}
}

func (mc *MeshCatalog) LocalEndpoints(w http.ResponseWriter, r *http.Request) {
	endpointMap, err := mc.ListLocalClusterEndpoints()
	if err != nil {
		log.Error().Msgf("err fetching endpoints %+v", err)
	}

	if err := json.NewEncoder(w).Encode(endpointMap); err != nil {
		log.Error().Msgf("err encoding endpoints %+v", err)
	}
}

func (mc *MeshCatalog) ApigroupMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" || r.Method == "DELETE" || r.Method == "PUT" {
		mc.witesandCatalog.UpdateApigroupMap(w, r.Method, r)
	} else {
		http.Error(w, "Invalid request method.", 405)
		return
	}
}

func (mc *MeshCatalog) ListLocalGatewaypods(svcName string) ([]string, error) {
	kubeClient := mc.kubeClient
	podList, err := kubeClient.CoreV1().Pods("default").List(context.Background(), v12.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error listing pods in namespace %s", "default")
		return nil, fmt.Errorf("error listing pod")
	}

	searchList := make([]string, 0)
	for _, pod := range podList.Items {
		if strings.Contains(pod.Name, svcName) && pod.Status.Phase == "Running" {
			log.Info().Msgf("pod.Name=%+v, pod.status=%+v \n", pod.Name, pod.Status.Phase)
			searchList = append(searchList, pod.Name)
		}
	}
	return searchList, nil
}

func (mc *MeshCatalog) ListAllGatewaypods(svcName string) ([]string, error) {
	searchList, _ := mc.ListLocalGatewaypods(svcName)

	// Add from remote pods from Remote osm-controller
	remoteProvider := mc.GetProvider("Remote")
	if remoteProvider != nil  {
		svc := service.MeshService{
			Namespace: "default",
			Name:      svcName,
		}
		// Note this is Service specific instead of pod specific.
		eps := remoteProvider.ListEndpointsForService(svc)
		if len(eps) > 0 {
			searchList = append(searchList, svcName)
		}
	} else {
		log.Info().Msgf("[GetGatewaypods]: Remote provider is nil")
	}
	return searchList, nil
}
