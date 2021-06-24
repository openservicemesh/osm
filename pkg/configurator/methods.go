package configurator

import (
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/auth"
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

func marshalConfigToJSON(config *v1alpha1.MeshConfigSpec) (string, error) {
	bytes, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// GetMeshConfigJSON returns the MeshConfig in pretty JSON.
func (c *Client) GetMeshConfigJSON() (string, error) {
	cm, err := marshalConfigToJSON(&c.getMeshConfig().Spec)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling MeshConfig %s: %+v", c.getMeshConfigCacheKey(), c.getMeshConfig())
		return "", err
	}
	return cm, nil
}

// IsPermissiveTrafficPolicyMode tells us whether the OSM Control Plane is in permissive mode,
// where all existing traffic is allowed to flow as it is,
// or it is in SMI Spec mode, in which only traffic between source/destinations
// referenced in SMI policies is allowed.
func (c *Client) IsPermissiveTrafficPolicyMode() bool {
	return c.getMeshConfig().Spec.Traffic.EnablePermissiveTrafficPolicyMode
}

// IsEgressEnabled determines whether egress is globally enabled in the mesh or not.
func (c *Client) IsEgressEnabled() bool {
	return c.getMeshConfig().Spec.Traffic.EnableEgress
}

// IsDebugServerEnabled determines whether osm debug HTTP server is enabled
func (c *Client) IsDebugServerEnabled() bool {
	return c.getMeshConfig().Spec.Observability.EnableDebugServer
}

// IsTracingEnabled returns whether tracing is enabled
func (c *Client) IsTracingEnabled() bool {
	return c.getMeshConfig().Spec.Observability.Tracing.Enable
}

// GetTracingHost is the host to which we send tracing spans
func (c *Client) GetTracingHost() string {
	tracingAddress := c.getMeshConfig().Spec.Observability.Tracing.Address
	if tracingAddress != "" {
		return tracingAddress
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", constants.DefaultTracingHost, c.GetOSMNamespace())
}

// GetTracingPort returns the tracing listener port
func (c *Client) GetTracingPort() uint32 {
	tracingPort := c.getMeshConfig().Spec.Observability.Tracing.Port
	if tracingPort != 0 {
		return uint32(tracingPort)
	}
	return constants.DefaultTracingPort
}

// GetTracingEndpoint returns the listener's collector endpoint
func (c *Client) GetTracingEndpoint() string {
	tracingEndpoint := c.getMeshConfig().Spec.Observability.Tracing.Endpoint
	if tracingEndpoint != "" {
		return tracingEndpoint
	}
	return constants.DefaultTracingEndpoint
}

// UseHTTPSIngress determines whether traffic between ingress and backend pods should use HTTPS protocol
func (c *Client) UseHTTPSIngress() bool {
	return c.getMeshConfig().Spec.Traffic.UseHTTPSIngress
}

// GetMaxDataPlaneConnections returns the max data plane connections allowed, 0 if disabled
func (c *Client) GetMaxDataPlaneConnections() int {
	return c.getMeshConfig().Spec.Sidecar.MaxDataPlaneConnections
}

// GetEnvoyLogLevel returns the envoy log level
func (c *Client) GetEnvoyLogLevel() string {
	logLevel := c.getMeshConfig().Spec.Sidecar.LogLevel
	if logLevel != "" {
		return logLevel
	}
	return constants.DefaultEnvoyLogLevel
}

// GetEnvoyImage returns the envoy image
func (c *Client) GetEnvoyImage() string {
	image := c.getMeshConfig().Spec.Sidecar.EnvoyImage
	if image != "" {
		return image
	}
	return constants.DefaultEnvoyImage
}

// GetInitContainerImage returns the init container image
func (c *Client) GetInitContainerImage() string {
	initImage := c.getMeshConfig().Spec.Sidecar.InitContainerImage
	if initImage != "" {
		return initImage
	}
	return constants.DefaultInitContainerImage
}

// GetServiceCertValidityPeriod returns the validity duration for service certificates, and a default in case of invalid duration
func (c *Client) GetServiceCertValidityPeriod() time.Duration {
	durationStr := c.getMeshConfig().Spec.Certificate.ServiceCertValidityDuration
	validityDuration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing service certificate validity duration %s", durationStr)
		return defaultServiceCertValidityDuration
	}

	return validityDuration
}

// GetOutboundIPRangeExclusionList returns the list of IP ranges of the form x.x.x.x/y to exclude from outbound sidecar interception
func (c *Client) GetOutboundIPRangeExclusionList() []string {
	return c.getMeshConfig().Spec.Traffic.OutboundIPRangeExclusionList
}

// GetOutboundPortExclusionList returns the list of ports (positive integers) to exclude from outbound sidecar interception
func (c *Client) GetOutboundPortExclusionList() []int {
	return c.getMeshConfig().Spec.Traffic.OutboundPortExclusionList
}

// GetInboundPortExclusionList returns the list of ports (positive integers) to exclude from inbound sidecar interception
func (c *Client) GetInboundPortExclusionList() []int {
	return c.getMeshConfig().Spec.Traffic.InboundPortExclusionList
}

// IsPrivilegedInitContainer returns whether init containers should be privileged
func (c *Client) IsPrivilegedInitContainer() bool {
	return c.getMeshConfig().Spec.Sidecar.EnablePrivilegedInitContainer
}

// GetConfigResyncInterval returns the duration for resync interval.
// If error or non-parsable value, returns 0 duration
func (c *Client) GetConfigResyncInterval() time.Duration {
	resyncDuration := c.getMeshConfig().Spec.Sidecar.ConfigResyncInterval
	duration, err := time.ParseDuration(resyncDuration)
	if err != nil {
		log.Debug().Err(err).Msgf("Error parsing config resync interval: %s", duration)
		return time.Duration(0)
	}
	return duration
}

// GetProxyResources returns the `Resources` configured for proxies, if any
func (c *Client) GetProxyResources() corev1.ResourceRequirements {
	return c.getMeshConfig().Spec.Sidecar.Resources
}

// GetInboundExternalAuthConfig returns the External Authentication configuration for incoming traffic, if any
func (c *Client) GetInboundExternalAuthConfig() auth.ExtAuthConfig {
	extAuthConfig := auth.ExtAuthConfig{}
	inboundExtAuthzMeshConfig := c.getMeshConfig().Spec.Traffic.InboundExternalAuthorization

	extAuthConfig.Enable = inboundExtAuthzMeshConfig.Enable
	extAuthConfig.Address = inboundExtAuthzMeshConfig.Address
	extAuthConfig.Port = uint16(inboundExtAuthzMeshConfig.Port)
	extAuthConfig.StatPrefix = inboundExtAuthzMeshConfig.StatPrefix
	extAuthConfig.FailureModeAllow = inboundExtAuthzMeshConfig.FailureModeAllow

	duration, err := time.ParseDuration(inboundExtAuthzMeshConfig.Timeout)
	if err != nil {
		log.Debug().Err(err).Msgf("ExternAuthzTimeout: Not a valid duration %s. defaulting to 1s.", duration)
		duration = 1 * time.Second
	}
	extAuthConfig.AuthzTimeout = duration

	return extAuthConfig
}

// GetFeatureFlags returns OSM's feature flags
func (c *Client) GetFeatureFlags() v1alpha1.FeatureFlags {
	return c.getMeshConfig().Spec.FeatureFlags
}
