package debugger

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type policies struct {
	TrafficSplits   []string `json:"traffic_splits"`
	SplitServices   []string `json:"split_services"`
	ServiceAccounts []string `json:"service_accounts"`
	TrafficSpecs    []string `json:"traffic_specs"`
	TrafficTargets  []string `json:"traffic_targets"`
	Services        []string `json:"services"`
}

func (ds debugServer) getSMIPoliciesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trafficSplits, splitServices, serviceAccounts, trafficSpecs, trafficTargets, services := ds.meshCatalogDebugger.ListSMIPolicies()

		var p policies

		for _, item := range trafficSplits {
			p.TrafficSplits = append(p.TrafficSplits, item.Name)
		}

		for _, item := range splitServices {
			p.SplitServices = append(p.SplitServices, item.NamespacedService.String())
		}

		for _, item := range serviceAccounts {
			p.ServiceAccounts = append(p.ServiceAccounts, item.String())
		}

		for _, item := range trafficSpecs {
			p.TrafficSpecs = append(p.TrafficSpecs, item.Name)
		}

		for _, item := range trafficTargets {
			p.TrafficTargets = append(p.TrafficTargets, item.Name)
		}

		for _, itme := range services {
			p.Services = append(p.Services, itme.Name)
		}

		jsonPolicies, err := json.Marshal(p)
		if err != nil {
			log.Error().Err(err).Msg("Error marshalling p")
		}

		_, _ = fmt.Fprint(w, string(jsonPolicies))
	})
}
