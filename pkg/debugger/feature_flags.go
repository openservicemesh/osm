package debugger

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/openservicemesh/osm/pkg/featureflags"
)

func (ds DebugConfig) getFeatureFlags() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if featureFlagsJSON, err := json.Marshal(featureflags.Features); err != nil {
			log.Error().Err(err).Msgf("Error marshaling feature flags struct: %+v", featureflags.Features)
		} else {
			_, _ = fmt.Fprint(w, string(featureFlagsJSON))
		}
	})
}
