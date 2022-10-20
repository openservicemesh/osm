package debugger

import (
	"encoding/json"
	"fmt"
	"net/http"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/utils"
)

type policies struct {
	TrafficSplits   []*split.TrafficSplit        `json:"traffic_splits"`
	ServiceAccounts []identity.K8sServiceAccount `json:"service_accounts"`
	RouteGroups     []*spec.HTTPRouteGroup       `json:"route_groups"`
	TrafficTargets  []*access.TrafficTarget      `json:"traffic_targets"`
}

func (ds DebugConfig) getOSMConfigHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		confJSON, err := utils.MeshConfigToJSON(ds.computeClient.GetMeshConfig())
		if err != nil {
			log.Error().Err(err).Msg("error getting MeshConfig JSON")
			return
		}
		_, _ = fmt.Fprint(w, confJSON)
	})
}

func (ds DebugConfig) getSMIPoliciesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p policies
		p.TrafficSplits = ds.computeClient.ListTrafficSplits()
		p.ServiceAccounts = ds.computeClient.ListServiceAccountsFromTrafficTargets()
		p.RouteGroups = ds.computeClient.ListHTTPTrafficSpecs()
		p.TrafficTargets = ds.computeClient.ListTrafficTargets()

		jsonPolicies, err := json.Marshal(p)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling policy %+v", p)
		}

		_, _ = fmt.Fprint(w, string(jsonPolicies))
	})
}
