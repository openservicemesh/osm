package remote

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/pkg/errors"

	a "github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/witesand"
)

const (
        // defaultAppProtocol is the default application protocol for a port if unspecified
        defaultAppProtocol = "http"
)

// NewProvider implements mesh.EndpointsProvider, which creates a new Kubernetes cluster/compute provider.
func NewProvider(kubeClient kubernetes.Interface, wsCatalog *witesand.WitesandCatalog, clusterId string, stop chan struct{}, meshSpec smi.MeshSpec, providerIdent string) (*Client, error) {

	client := Client{
		wsCatalog:           wsCatalog,
		providerIdent:       providerIdent,
		clusterId:           clusterId,
		meshSpec:            meshSpec,
		caches:              nil,
		announcements:       make(chan a.Announcement),
	}

	client.caches = &CacheCollection{
		k8sToServiceEndpoints: make(map[string]*ServiceToEndpointMap),
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

	for _, epMap := range c.caches.k8sToServiceEndpoints {
		if eps, exists := epMap.endpoints[svc.String()]; exists {
			log.Info().Msgf("[%s:ListEndpointsForService] Endpoints for service %s on Remote:%+v", c.providerIdent, svc.String(), eps)
			endpoints = append(endpoints, eps...)
		}
	}

	return endpoints
}

// GetServiceForServiceAccount retrieves the service for the given service account
func (c Client) GetServicesForServiceAccount(svcAccount service.K8sServiceAccount) ([]service.MeshService, error) {
	//log.Info().Msgf("[%s] Getting Services for service account %s on Remote", c.providerIdent, svcAccount)
	servicesSlice := make([]service.MeshService, 0)

	if c.caches == nil {
		return servicesSlice, errDidNotFindServiceForServiceAccount
	}

	svc := fmt.Sprintf("%s/%s", svcAccount.Namespace, svcAccount.Name)

	// TODO: is this needed

	for _, epMap := range c.caches.k8sToServiceEndpoints {
		if _, ok := epMap.endpoints[svc]; ok {
			namespacedService := service.MeshService{
				Namespace: svcAccount.Namespace,
				Name:      svcAccount.Name,
			}
			servicesSlice = append(servicesSlice, namespacedService)
			return servicesSlice, nil
		}
	}

	return servicesSlice, errDidNotFindServiceForServiceAccount
}

// GetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol
func (c Client) GetPortToProtocolMappingForService(svc service.MeshService) (map[uint32]string, error) {
	portToProtocolMap := make(map[uint32]string)

	// TODO

	return portToProtocolMap, nil
}

func (c *Client) GetResolvableEndpointsForService(svc service.MeshService) ([]endpoint.Endpoint, error) {
	eps := c.ListEndpointsForService(svc)
	if len(eps) == 0 {
		return nil, errServiceNotFound
	}
	return eps, nil
}

// GetAnnouncementsChannel returns the announcement channel for the Kubernetes endpoints provider.
func (c Client) GetAnnouncementsChannel() <-chan a.Announcement {
	return c.announcements
}

func (c *Client) run() error {

	// send HTTP request to remote OSM
	queryRemoteOsm := func(remoteOsmIP string) (*ServiceToEndpointMap, error) {
		log.Info().Msgf("[queryRemoteOsm] querying osm:%s", remoteOsmIP)
		dest := fmt.Sprintf("%s:%s", remoteOsmIP, witesand.HttpServerPort)
		url := fmt.Sprintf("http://%s/endpoints", dest)
		client := &http.Client{}
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set(witesand.HttpRemoteAddrHeader, c.wsCatalog.GetMyIP())
		req.Header.Set(witesand.HttpRemoteClusterIdHeader, c.clusterId)
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()

			serviceToEndpointMap := ServiceToEndpointMap{
				endpoints: make(map[string][]endpoint.Endpoint),
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(b, &serviceToEndpointMap.endpoints)
				log.Info().Msgf("[queryRemoteOsm] received response: %+v", serviceToEndpointMap.endpoints)
				if err == nil {
					return &serviceToEndpointMap, nil
				}
			}
		}
		log.Info().Msgf("[queryRemoteOsm] err:%+v", err)
		return nil, err
	}

	// update the cache
	updateCache := func(k8sName string, epMap *ServiceToEndpointMap) {
		log.Info().Msgf("[updateCache] updating %s", k8sName)
		c.caches.k8sToServiceEndpoints[k8sName] = epMap
	}

	poll := func() {
		log.Info().Msgf("[poll] started polling")
		ticker := time.NewTicker(15 * time.Second)
		for {
			<-ticker.C
			log.Info().Msgf("[poll] tick occurred")
			for clusterId, remoteK8s := range c.wsCatalog.ListRemoteK8s() {
				epMap, err := queryRemoteOsm(remoteK8s.OsmIP)
				if err == nil {
					updateCache(clusterId, epMap)
				}
			}
		}
	}

	// start an end-less loop
	go poll()

	return nil
}
