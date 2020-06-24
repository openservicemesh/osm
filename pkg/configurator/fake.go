package configurator

// FakeConfigurator is a fake namespace.Configurator used for testing
type FakeConfigurator struct {
	namespaces []string
	Configurator
}

// NewFakeConfigurator creates a fake configurator.
func NewFakeConfigurator() *ConfigMapWatcher {
	return &ConfigMapWatcher{
		osmNamespace:  "test-osm-namespace",
		announcements: make(chan interface{}),
		configMap: &configMap{
			LogVerbosity: "trace",
		},
	}
}
