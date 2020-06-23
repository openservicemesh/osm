package configurator

// FakeConfigurator is a fake namespace.Configurator used for testing
type FakeConfigurator struct {
	monitoredConfigurators []string
	Configurator
}

// NewFakeConfigurator creates a fake configurator.
func NewFakeConfigurator(monitoredConfigurators []string) FakeConfigurator {
	return FakeConfigurator{
		monitoredConfigurators: monitoredConfigurators,
	}
}

// IsMonitoredNamespace returns if the namespace is monitored
func (f FakeConfigurator) IsMonitoredNamespace(namespace string) bool {
	log.Debug().Msgf("Monitored namespaces = %v", f.monitoredConfigurators)
	for _, ns := range f.monitoredConfigurators {
		if ns == namespace {
			return true
		}
	}
	return false
}
