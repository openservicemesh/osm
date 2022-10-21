package debugger

import (
	"fmt"
	"net/http"

	"github.com/openservicemesh/osm/pkg/utils"
)

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
