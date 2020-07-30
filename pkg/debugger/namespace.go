package debugger

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/open-service-mesh/osm/pkg/service"
)

type namespace struct {
	Namespaces    []string            `json:"namespaces"`
}

func (ds debugServer) getMonitoredNamespacesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var n namespaces
		n.Namespaces = ds.meshCatalogDebugger.ListMonitoredNamespaces()

		jsonPolicies, err := json.Marshal(p)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling policy %+v", p)
		}

		_, _ = fmt.Fprint(w, string(jsonPolicies))
	})
}
