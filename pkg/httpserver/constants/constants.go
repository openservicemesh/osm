package constants

// OSM HTTP Server Paths
const (
	HealthReadinessPath = "/health/ready"
	HealthLivenessPath  = "/health/alive"
	MetricsPath         = "/metrics"
	VersionPath         = "/version"
	SmiVersionPath      = "/smi/version"
)

// OSM HTTP Server Responses
const (
	ServiceReadyResponse = "Service is ready"
	ServiceAliveResponse = "Service is alive"
)
