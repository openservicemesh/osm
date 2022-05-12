package debugger

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (ds DebugConfig) getFeatureFlags() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		featureFlags := ds.configurator.GetFeatureFlags()
		if featureFlagsJSON, err := json.Marshal(featureFlags); err != nil {
			log.Error().Err(err).Msgf("Error marshaling feature flags struct: %+v", featureFlags)
		} else {
			_, _ = fmt.Fprint(w, string(featureFlagsJSON))
		}
	})
}
