package debugger

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha2"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-service-mesh/osm/pkg/service"
)

type policies struct {
	TrafficSplits    []*split.TrafficSplit              `json:"traffic_splits"`
	WeightedServices []service.WeightedService          `json:"weighted_services"`
	ServiceAccounts  []service.NamespacedServiceAccount `json:"service_accounts"`
	RouteGroups      []*spec.HTTPRouteGroup             `json:"route_groups"`
	TrafficTargets   []*target.TrafficTarget            `json:"traffic_targets"`
	Services         []*corev1.Service                  `json:"services"`
}

type Config struct {
	LogVerbosity []string `json:"log_verbosity"`
	Namespaces   []string `json:"namespaces"`
	Ingresses    []string `json:"ingresses"`
}

func (ds debugServer) getOSMConfig() *Config {
	filename := "/etc/config/osm.conf"
	configData, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading OSM config file %s", filename)
	}

	conf := Config{}
	err = yaml.Unmarshal(configData, &conf)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling file %s with content %s", filename, string(configData))
	}

	return &conf
}

func (ds debugServer) getOSMConfigHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conf := ds.getOSMConfig()
		confJSON, err := json.MarshalIndent(conf, "", "    ")
		if err != nil {
			log.Error().Err(err)
			return
		}
		_, _ = fmt.Fprint(w, string(confJSON))
	})
}

func (ds debugServer) getSMIPoliciesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var p policies
		p.TrafficSplits, p.WeightedServices, p.ServiceAccounts, p.RouteGroups, p.TrafficTargets, p.Services = ds.meshCatalogDebugger.ListSMIPolicies()

		jsonPolicies, err := json.Marshal(p)
		if err != nil {
			log.Error().Err(err).Msg("Error marshalling p")
		}

		_, _ = fmt.Fprint(w, string(jsonPolicies))
	})
}
