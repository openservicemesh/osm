package version

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("version")

// BuildDate is the date when the binary was built
var BuildDate string

// GitCommit is the commit hash when the binary was built
var GitCommit string

// Version is the version of the compiled software
var Version string

// Info is a struct helpful for JSON serialization of the OSM Controller version information.
type Info struct {
	// Version is the version of the OSM Controller.
	Version string `json:"version,omitempty"`

	// GitCommit is the git commit hash of the OSM Controller.
	GitCommit string `json:"git_commit,omitempty"`

	// BuildDate is the build date of the OSM Controller.
	BuildDate string `json:"build_date,omitempty"`
}

// GetVersionHandler returns an HTTP handler that returns the version info
func GetVersionHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		versionInfo := Info{
			Version:   Version,
			BuildDate: BuildDate,
			GitCommit: GitCommit,
		}

		if jsonVersionInfo, err := json.Marshal(versionInfo); err != nil {
			log.Error().Err(err).Msgf("Error marshaling version info struct: %+v", versionInfo)
		} else {
			_, _ = fmt.Fprint(w, string(jsonVersionInfo))
		}
	})
}
