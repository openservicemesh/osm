package configurator

import (
	"github.com/openservicemesh/osm/pkg/constants"
)

// FakeConfigurator is the fake type for the Configurator client
type FakeConfigurator struct {
	OSMNamespace                string
	PermissiveTrafficPolicyMode bool
	Egress                      bool
	PrometheusScraping          bool
	TracingEnable               bool
	MeshCIDRRanges              []string
	HTTPSIngress                bool
}

// NewFakeConfigurator create a new fake Configurator
func NewFakeConfigurator() Configurator {
	return FakeConfigurator{
		Egress:             true,
		PrometheusScraping: true,
		TracingEnable:      true,
		HTTPSIngress:       false,
	}
}

// NewFakeConfiguratorWithOptions create a new fake Configurator
func NewFakeConfiguratorWithOptions(f FakeConfigurator) Configurator {
	return FakeConfigurator{
		OSMNamespace:                f.OSMNamespace,
		PermissiveTrafficPolicyMode: f.PermissiveTrafficPolicyMode,
		Egress:                      f.Egress,
		PrometheusScraping:          f.PrometheusScraping,
		TracingEnable:               f.TracingEnable,
		MeshCIDRRanges:              f.MeshCIDRRanges,
		HTTPSIngress:                f.HTTPSIngress,
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

// IsTracingEnabled determines whether tracing is enabled
func (f FakeConfigurator) IsTracingEnabled() bool {
	return f.TracingEnable
}

// GetMeshCIDRRanges returns a list of mesh CIDR ranges
func (f FakeConfigurator) GetMeshCIDRRanges() []string {
	return f.MeshCIDRRanges
}

// UseHTTPSIngress determines whether we use HTTPS for ingress to backend pods traffic
func (f FakeConfigurator) UseHTTPSIngress() bool {
	return f.HTTPSIngress
}

// GetAnnouncementsChannel returns a fake announcement channel
func (f FakeConfigurator) GetAnnouncementsChannel() <-chan interface{} {
	return make(chan interface{})
}

// GetTracingHost is the host to which we send tracing spans
func (f FakeConfigurator) GetTracingHost() string {
	return constants.DefaultTracingHost
}

// GetTracingPort returns the listener port
func (f FakeConfigurator) GetTracingPort() uint32 {
	return constants.DefaultTracingPort
}

// GetTracingEndpoint returns the listener's collector endpoint
func (f FakeConfigurator) GetTracingEndpoint() string {
	return constants.DefaultTracingEndpoint
}

// GetEnvoyLogLevel returns the OSM log level
func (f FakeConfigurator) GetEnvoyLogLevel() string {
	return constants.DefaultEnvoyLogLevel
}
