package version

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
