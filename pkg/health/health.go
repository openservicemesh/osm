package health

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/openservicemesh/osm/pkg/version"
)

// Probe is a type alias for a function.
type Probe func() bool

// Probes is the interface for liveness and readiness probes
type Probes interface {
	Liveness() bool
	Readiness() bool
}

func makeHandler(probe Probe) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(map[bool]int{
			true:  http.StatusOK,
			false: http.StatusServiceUnavailable,
		}[probe()])

		versionInfo := version.Info{
			Version:   version.Version,
			BuildDate: version.BuildDate,
			GitCommit: version.GitCommit,
		}

		if jsonVersionInfo, err := json.Marshal(versionInfo); err != nil {
			log.Error().Err(err).Msgf("Error marshaling version info struct: %+v", versionInfo)
		} else {
			_, _ = fmt.Fprint(w, string(jsonVersionInfo))
		}
	})
}

// ReadinessHandler returns readiness http handlers for health
func ReadinessHandler(probe Probes) http.Handler {
	return makeHandler(probe.Readiness)
}

// LivenessHandler returns readiness http handlers for health
func LivenessHandler(probe Probes) http.Handler {
	return makeHandler(probe.Liveness)
}
