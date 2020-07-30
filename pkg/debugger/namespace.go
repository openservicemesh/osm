package debugger

import (
	"encoding/json"
	"fmt"
	"net/http"
<<<<<<< HEAD
)

type namespaces struct {
=======

	"github.com/open-service-mesh/osm/pkg/service"
)

type namespace struct {
>>>>>>> fixed namespace imports with new openservicemesh label (removed dashes)
	Namespaces    []string            `json:"namespaces"`
}

func (ds debugServer) getMonitoredNamespacesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var n namespaces
		n.Namespaces = ds.meshCatalogDebugger.ListMonitoredNamespaces()

<<<<<<< HEAD
		jsonPolicies, err := json.Marshal(n)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling policy %+v", n)
=======
		jsonPolicies, err := json.Marshal(p)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling policy %+v", p)
>>>>>>> fixed namespace imports with new openservicemesh label (removed dashes)
		}

		_, _ = fmt.Fprint(w, string(jsonPolicies))
	})
}
