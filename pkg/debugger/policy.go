package debugger

import (
	"encoding/json"
	"fmt"
	"net/http"

	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha2"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/service"
)

type policies struct {
	TrafficSplits    []*split.TrafficSplit       `json:"traffic_splits"`
	WeightedServices []service.WeightedService   `json:"weighted_services"`
	ServiceAccounts  []service.K8sServiceAccount `json:"service_accounts"`
	RouteGroups      []*spec.HTTPRouteGroup      `json:"route_groups"`
	TrafficTargets   []*target.TrafficTarget     `json:"traffic_targets"`
	Services         []*corev1.Service           `json:"services"`
}

func (ds debugServer) getSMIPoliciesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var p policies
		p.TrafficSplits, p.WeightedServices, p.ServiceAccounts, p.RouteGroups, p.TrafficTargets, p.Services = ds.meshCatalogDebugger.ListSMIPolicies()

		jsonPolicies, err := json.Marshal(p)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling policy %+v", p)
		}

		_, _ = fmt.Fprint(w, string(jsonPolicies))
	})
}
