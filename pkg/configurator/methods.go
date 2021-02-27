package configurator

import (
	"encoding/json"
	"errors"
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
func (c *Client) GetOSMNamespace() string {
	return c.osmNamespace
}

func marshalConfigToJSON(config *osmConfig) ([]byte, error) {
	return json.MarshalIndent(config, "", "    ")
}

// GetConfigMap returns the ConfigMap in pretty JSON.
func (c *Client) GetConfigMap() ([]byte, error) {
	if c.getConfigMap() == nil {
		return nil, errors.New("no ConfigMap")
	}
	cm, err := marshalConfigToJSON(c.getConfigMap())
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling ConfigMap %s: %+v", c.getConfigMapCacheKey(), c.getConfigMap())
		return nil, err
	}
	return cm, nil
}

// IsPermissiveTrafficPolicyMode tells us whether the OSM Control Plane is in permissive mode,
// where all existing traffic is allowed to flow as it is,
// or it is in SMI Spec mode, in which only traffic between source/destinations
// referenced in SMI policies is allowed.
func (c *Client) IsPermissiveTrafficPolicyMode() bool {
	if c.getConfigMap() == nil {
		return false
	}
	return c.getConfigMap().PermissiveTrafficPolicyMode
}

// IsEgressEnabled determines whether egress is globally enabled in the mesh or not.
func (c *Client) IsEgressEnabled() bool {
	if c.getConfigMap() == nil {
		return false
	}
	return c.getConfigMap().Egress
}

// IsDebugServerEnabled determines whether osm debug HTTP server is enabled
func (c *Client) IsDebugServerEnabled() bool {
	if c.getConfigMap() == nil {
		return false
	}
	return c.getConfigMap().EnableDebugServer
}

// IsPrometheusScrapingEnabled determines whether Prometheus is enabled for scraping metrics
func (c *Client) IsPrometheusScrapingEnabled() bool {
	if c.getConfigMap() == nil {
		return false
	}
	return c.getConfigMap().PrometheusScraping
}

// IsTracingEnabled returns whether tracing is enabled
func (c *Client) IsTracingEnabled() bool {
	if c.getConfigMap() == nil {
		return false
	}
	return c.getConfigMap().TracingEnable
}

// GetTracingHost is the host to which we send tracing spans
func (c *Client) GetTracingHost() string {
	if c.getConfigMap() == nil {
		return ""
	}
	tracingAddress := c.getConfigMap().TracingAddress
	if tracingAddress != "" {
		return tracingAddress
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", constants.DefaultTracingHost, c.GetOSMNamespace())
}

// GetTracingPort returns the tracing listener port
func (c *Client) GetTracingPort() uint32 {
	if c.getConfigMap() == nil {
		return 0
	}
	tracingPort := c.getConfigMap().TracingPort
	if tracingPort != 0 {
		return uint32(tracingPort)
	}
	return constants.DefaultTracingPort
}

// GetTracingEndpoint returns the listener's collector endpoint
func (c *Client) GetTracingEndpoint() string {
	if c.getConfigMap() == nil {
		return ""
	}
	tracingEndpoint := c.getConfigMap().TracingEndpoint
	if tracingEndpoint != "" {
		return tracingEndpoint
	}
	return constants.DefaultTracingEndpoint
}

// UseHTTPSIngress determines whether traffic between ingress and backend pods should use HTTPS protocol
func (c *Client) UseHTTPSIngress() bool {
	if c.getConfigMap() == nil {
		return false
	}
	return c.getConfigMap().UseHTTPSIngress
}

// GetEnvoyLogLevel returns the envoy log level
func (c *Client) GetEnvoyLogLevel() string {
	if c.getConfigMap() == nil {
		return constants.DefaultEnvoyLogLevel
	}
	logLevel := c.getConfigMap().EnvoyLogLevel
	if logLevel != "" {
		return logLevel
	}
	return constants.DefaultEnvoyLogLevel
}

// GetServiceCertValidityPeriod returns the validity duration for service certificates, and a default in case of invalid duration
func (c *Client) GetServiceCertValidityPeriod() time.Duration {
	if c.getConfigMap() == nil {
		return defaultServiceCertValidityDuration
	}
	durationStr := c.getConfigMap().ServiceCertValidityDuration
	validityDuration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing service certificate validity duration %s=%s", serviceCertValidityDurationKey, durationStr)
		return defaultServiceCertValidityDuration
	}

	return validityDuration
}

// GetOutboundIPRangeExclusionList returns the list of IP ranges of the form x.x.x.x/y to exclude from outbound sidecar interception
func (c *Client) GetOutboundIPRangeExclusionList() []string {
	if c.getConfigMap() == nil {
		return nil
	}
	ipRangesStr := c.getConfigMap().OutboundIPRangeExclusionList
	if ipRangesStr == "" {
		return nil
	}

	exclusionList := strings.Split(ipRangesStr, ",")
	for i := range exclusionList {
		exclusionList[i] = strings.TrimSpace(exclusionList[i])
	}

	return exclusionList
}

// IsPrivilegedInitContainer returns whether init containers should be privileged
func (c *Client) IsPrivilegedInitContainer() bool {
	if c.getConfigMap() == nil {
		return false
	}
	return c.getConfigMap().EnablePrivilegedInitContainer
}
