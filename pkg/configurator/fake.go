package configurator

// FakeConfigurator is the fake type for the Configurator client
type FakeConfigurator struct {
	OSMNamespace                string
	PermissiveTrafficPolicyMode bool
	Egress                      bool
	PrometheusScraping          bool
	ZipkinTracing               bool
}

// NewFakeConfigurator create a new fake Configurator
func NewFakeConfigurator() Configurator {
	return FakeConfigurator{
		Egress:             true,
		PrometheusScraping: true,
		ZipkinTracing:      true,
	}
}

// GetConfigMap returns the data stored in the configMap
func (f FakeConfigurator) GetConfigMap() ([]byte, error) {
	return nil, nil
}

// GetOSMNamespace returns the namespace in which the OSM controller pod resides.
func (f FakeConfigurator) GetOSMNamespace() string {
	return f.OSMNamespace
}

// IsPermissiveTrafficPolicyMode tells us whether the OSM Control Plane is in permissive mode
func (f FakeConfigurator) IsPermissiveTrafficPolicyMode() bool {
	return f.PermissiveTrafficPolicyMode
}

// IsEgressEnabled determines whether egress is globally enabled in the mesh or not.
func (f FakeConfigurator) IsEgressEnabled() bool {
	return f.Egress
}

// IsPrometheusScrapingEnabled determines whether Prometheus is enabled for scraping metrics
func (f FakeConfigurator) IsPrometheusScrapingEnabled() bool {
	return f.PrometheusScraping
}

// IsZipkinTracingEnabled determines whether Zipkin tracing is enabled
func (f FakeConfigurator) IsZipkinTracingEnabled() bool {
	return f.ZipkinTracing
}

// GetAnnouncementsChannel returns a fake announcement channel
func (f FakeConfigurator) GetAnnouncementsChannel() <-chan interface{} {
	return make(chan interface{})
}
