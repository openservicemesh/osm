package debugger

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type namespaces struct {
	Namespaces []string `json:"namespaces"`
}

func (ds DebugConfig) getMonitoredNamespacesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var n namespaces
		n.Namespaces = ds.meshCatalogDebugger.ListMonitoredNamespaces()

		jsonPolicies, err := json.Marshal(n)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling policy %+v", n)
		}

		_, _ = fmt.Fprint(w, string(jsonPolicies))
	})
}
