package configurator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	// defaultServiceCertValidityDuration is the default validity duration for service certificates
	defaultServiceCertValidityDuration = 24 * time.Hour
)

// The functions in this file implement the configurator.Configurator interface

// GetOSMNamespace returns the namespace in which the OSM controller pod resides.
func (c *CRDClient) GetOSMNamespace() string {
	return c.osmNamespace
}

func marshalConfigToJSON(config *osmConfig) ([]byte, error) {
	return json.MarshalIndent(config, "", "    ")
}

// GetMeshConfigJSON returns the MeshConfig in pretty JSON.
func (c *CRDClient) GetMeshConfigJSON() ([]byte, error) {
	cm, err := marshalConfigToJSON(c.getMeshConfig())
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling MeshConfig %s: %+v", c.getMeshConfigCacheKey(), c.getMeshConfig())
		return nil, err
	}
	return cm, nil
}

// IsPermissiveTrafficPolicyMode tells us whether the OSM Control Plane is in permissive mode,
// where all existing traffic is allowed to flow as it is,
// or it is in SMI Spec mode, in which only traffic between source/destinations
// referenced in SMI policies is allowed.
func (c *CRDClient) IsPermissiveTrafficPolicyMode() bool {
	return c.getMeshConfig().PermissiveTrafficPolicyMode
}

// IsEgressEnabled determines whether egress is globally enabled in the mesh or not.
func (c *CRDClient) IsEgressEnabled() bool {
	return c.getMeshConfig().Egress
}

// IsDebugServerEnabled determines whether osm debug HTTP server is enabled
func (c *CRDClient) IsDebugServerEnabled() bool {
	return c.getMeshConfig().EnableDebugServer
}

// IsPrometheusScrapingEnabled determines whether Prometheus is enabled for scraping metrics
func (c *CRDClient) IsPrometheusScrapingEnabled() bool {
	return c.getMeshConfig().PrometheusScraping
}

// IsTracingEnabled returns whether tracing is enabled
func (c *CRDClient) IsTracingEnabled() bool {
	return c.getMeshConfig().TracingEnable
}

// GetTracingHost is the host to which we send tracing spans
func (c *CRDClient) GetTracingHost() string {
	tracingAddress := c.getMeshConfig().TracingAddress
	if tracingAddress != "" {
		return tracingAddress
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", constants.DefaultTracingHost, c.GetOSMNamespace())
}

// GetTracingPort returns the tracing listener port
func (c *CRDClient) GetTracingPort() uint32 {
	tracingPort := c.getMeshConfig().TracingPort
	if tracingPort != 0 {
		return uint32(tracingPort)
	}
	return constants.DefaultTracingPort
}

// GetTracingEndpoint returns the listener's collector endpoint
func (c *CRDClient) GetTracingEndpoint() string {
	tracingEndpoint := c.getMeshConfig().TracingEndpoint
	if tracingEndpoint != "" {
		return tracingEndpoint
	}
	return constants.DefaultTracingEndpoint
}

// UseHTTPSIngress determines whether traffic between ingress and backend pods should use HTTPS protocol
func (c *CRDClient) UseHTTPSIngress() bool {
	return c.getMeshConfig().UseHTTPSIngress
}

// GetMaxDataPlaneConnections returns the max data plane connections allowed, 0 if disabled
func (c *CRDClient) GetMaxDataPlaneConnections() int {
	return c.getMeshConfig().MaxDataPlaneConnections
}

// GetEnvoyLogLevel returns the envoy log level
func (c *CRDClient) GetEnvoyLogLevel() string {
	logLevel := c.getMeshConfig().EnvoyLogLevel
	if logLevel != "" {
		return logLevel
	}
	return constants.DefaultEnvoyLogLevel
}

// GetEnvoyImage returns the envoy image
func (c *CRDClient) GetEnvoyImage() string {
	image := c.getMeshConfig().EnvoyImage
	if image != "" {
		return image
	}
	return constants.DefaultEnvoyImage
}

// GetServiceCertValidityPeriod returns the validity duration for service certificates, and a default in case of invalid duration
func (c *CRDClient) GetServiceCertValidityPeriod() time.Duration {
	durationStr := c.getMeshConfig().ServiceCertValidityDuration
	validityDuration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing service certificate validity duration %s=%s", serviceCertValidityDurationKey, durationStr)
		return defaultServiceCertValidityDuration
	}

	return validityDuration
}

// GetOutboundIPRangeExclusionList returns the list of IP ranges of the form x.x.x.x/y to exclude from outbound sidecar interception
func (c *CRDClient) GetOutboundIPRangeExclusionList() []string {
	ipRangesStr := c.getMeshConfig().OutboundIPRangeExclusionList
	if ipRangesStr == "" {
		return nil
	}

	exclusionList := strings.Split(ipRangesStr, ",")
	for i := range exclusionList {
		exclusionList[i] = strings.TrimSpace(exclusionList[i])
	}

	return exclusionList
}

// GetOutboundPortExclusionList returns the list of ports (positive integers) to exclude from outbound sidecar interception
func (c *CRDClient) GetOutboundPortExclusionList() []string {
	portsStr := c.getMeshConfig().OutboundPortExclusionList
	if portsStr == "" {
		return nil
	}

	portExclusionList := strings.Split(portsStr, ",")
	for i := range portExclusionList {
		portExclusionList[i] = strings.TrimSpace(portExclusionList[i])
	}

	return portExclusionList
}

// IsPrivilegedInitContainer returns whether init containers should be privileged
func (c *CRDClient) IsPrivilegedInitContainer() bool {
	return c.getMeshConfig().EnablePrivilegedInitContainer
}

// GetConfigResyncInterval returns the duration for resync interval.
// If error or non-parsable value, returns 0 duration
func (c *CRDClient) GetConfigResyncInterval() time.Duration {
	resyncDuration := c.getMeshConfig().ConfigResyncInterval
	duration, err := time.ParseDuration(resyncDuration)
	if err != nil {
		log.Debug().Err(err).Msgf("Error parsing config resync interval: %s", duration)
		return time.Duration(0)
	}
	return duration
}

// GetPermissiveTrafficPolicyMode returns the duration for resync interval.
// If error or non-parsable value, returns 0 duration
func (c *CRDClient) GetPermissiveTrafficPolicyMode() bool {
	return c.getMeshConfig().PermissiveTrafficPolicyMode
}
