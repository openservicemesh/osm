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
	ZipkinTracing               bool
	MeshCIDRRanges              []string
	HTTPSIngress                bool
}

// NewFakeConfigurator create a new fake Configurator
func NewFakeConfigurator() Configurator {
	return FakeConfigurator{
		Egress:             true,
		PrometheusScraping: true,
		ZipkinTracing:      true,
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
		ZipkinTracing:               f.ZipkinTracing,
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

// IsZipkinTracingEnabled determines whether Zipkin tracing is enabled
func (f FakeConfigurator) IsZipkinTracingEnabled() bool {
	return f.ZipkinTracing
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

// GetZipkinHost is the host to which we send Zipkin spanspkg/envoy/cds/response.go
func (f FakeConfigurator) GetZipkinHost() string {
	return constants.DefaultZipkinAddress
}

// GetZipkinPort returns the Zipkin port
func (f FakeConfigurator) GetZipkinPort() uint32 {
	return constants.DefaultZipkinPort
}

// GetZipkinEndpoint returns the Zipkin endpoint
func (f FakeConfigurator) GetZipkinEndpoint() string {
	return constants.DefaultZipkinEndpoint
}

// GetEnvoyLogLevel returns the Zipkin endpoint
func (f FakeConfigurator) GetEnvoyLogLevel() string {
	return constants.DefaultEnvoyLogLevel
}
