package configurator

// Config is a struct with common global config settings.
type Config struct {
	// OSMNamespace is the Kubernetes namespace in which the OSM controller is installed.
	OSMNamespace string

	// EnablePrometheus enables Prometheus metrics integration when true
	EnablePrometheus bool

	// EnableTracing enables Zipkin tracing when true.
	EnableTracing bool
}
