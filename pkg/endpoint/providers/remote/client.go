package remote

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
)

// NewProvider implements mesh.EndpointsProvider, which creates a new Kubernetes cluster/compute provider.
func NewProvider(kubeClient kubernetes.Interface, clusterId string, stop chan struct{}, meshSpec smi.MeshSpec, providerIdent string, remoteOsm string) (*Client, error) {

	client := Client{
		providerIdent:       providerIdent,
		clusterId:           clusterId,
		meshSpec:            meshSpec,
		caches:              nil,
		announcements:       make(chan interface{}),
		remoteOsm:           remoteOsm,
	}

	if err := client.run(); err != nil {
		return nil, errors.Errorf("Failed to start Remote EndpointProvider client: %+v", err)
	}
	log.Info().Msgf("[NewProvider] started Remote provider")

	return &client, nil
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interface: EndpointsProvider
func (c *Client) GetID() string {
	return c.providerIdent
}

// ListEndpointsForService retrieves the list of IP addresses for the given service
func (c Client) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	log.Info().Msgf("[%s] Getting Endpoints for service %s on Remote", c.providerIdent, svc)
	var endpoints []endpoint.Endpoint = []endpoint.Endpoint{}

	if c.caches == nil {
		return endpoints
	}

	if eps, ok := c.caches.endpoints[svc.String()]; ok {
		log.Info().Msgf("[%s] Endpoints for service %s on Remote:%+v", c.providerIdent, svc.String(), eps)
		return eps
	}
	log.Info().Msgf("[%s] No Endpoints for service %s on Remote", c.providerIdent)

	return endpoints
}

// GetServiceForServiceAccount retrieves the service for the given service account
func (c Client) GetServiceForServiceAccount(svcAccount service.K8sServiceAccount) (service.MeshService, error) {
	log.Info().Msgf("[%s] Getting Services for service account %s on Remote", c.providerIdent, svcAccount)

	if c.caches == nil {
		return service.MeshService{}, errDidNotFindServiceForServiceAccount
	}

	svc := fmt.Sprintf("%s/%s", svcAccount.Namespace, svcAccount.Name)

	if _, ok := c.caches.endpoints[svc]; ok {
		namespacedService := service.MeshService{
			Namespace: svcAccount.Namespace,
			Name:      svcAccount.Name,
		}
		return namespacedService, nil
	}

	return service.MeshService{}, errDidNotFindServiceForServiceAccount
}

// GetAnnouncementsChannel returns the announcement channel for the Kubernetes endpoints provider.
func (c Client) GetAnnouncementsChannel() <-chan interface{} {
	return c.announcements
}

func (c *Client) run() error {
	// convert catalog.endpoint to endpoint.Endpoint
	convertEndpoints := func(cataEps map[string][]catalog.EndpointJSON) *map[string][]endpoint.Endpoint {
		endpointMap := make(map[string][]endpoint.Endpoint)

		for svc, cataEpList := range cataEps {
			var eps = []endpoint.Endpoint{}
			for _, cataEp := range cataEpList {
				ep := endpoint.Endpoint{
					IP:   cataEp.IP,
					Port: cataEp.Port,
				}
				eps = append(eps, ep)
			}
			endpointMap[svc] = eps
		}
		return &endpointMap
	}

	// Get preferred outbound ip of this machine
	myIP := func(destIPPort string) string {
		conn, err := net.Dial("udp", destIPPort)
		if err != nil {
			return ""
		}
		defer conn.Close()

		localAddr := conn.LocalAddr().(*net.UDPAddr)

		return localAddr.IP.String()
	}

	// send HTTP request to remote OSM
	queryRemoteOsm := func() (*map[string][]endpoint.Endpoint, error) {
		log.Info().Msgf("[queryRemoteOsm] querying osm:%s", c.remoteOsm)
		dest := fmt.Sprintf("%s:2500", c.remoteOsm)
		url := fmt.Sprintf("http://%s/endpoints", dest)
		client := &http.Client{}
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("X-Osm-Origin-Ip", myIP(dest))
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			cataEndpointMap := make(map[string][]catalog.EndpointJSON)
			b, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(b, &cataEndpointMap)
				log.Info().Msgf("[queryRemoteOsm] received response: %+v", cataEndpointMap)
				if err == nil {
					endpointMap := convertEndpoints(cataEndpointMap)
					log.Info().Msgf("[queryRemoteOsm] converted response: %+v", endpointMap)
					return endpointMap, nil
				}
			}
		}
		log.Info().Msgf("[queryRemoteOsm] err:%+v", err)
		return nil, err
	}

	// update the cache
	updateCache := func(epMap *map[string][]endpoint.Endpoint) {
		log.Info().Msgf("[updateCache] received response, len:%d", len(*epMap))
		newCache := CacheCollection {
			endpoints: *epMap,
		}
		c.caches = &newCache
	}

	poll := func() {
		log.Info().Msgf("[poll] started polling")
		ticker := time.NewTicker(15 * time.Second)
		for {
			<-ticker.C
			log.Info().Msgf("[poll] tick occurred")
			if c.remoteOsm == "" {
				log.Info().Msgf("[poll] remoteOsmIP not set, yield")
				continue
			}
			epMap, err := queryRemoteOsm()
			if err == nil {
				updateCache(epMap)
			}
		}
	}

	// start an end-less loop
	go poll()

	return nil
}

func (c *Client) RegisterRemoteOSM(remote string) {
	log.Info().Msgf("[RegisterRemoteOSM] registering remote OSM:%s", remote)
	c.remoteOsm = remote
}
