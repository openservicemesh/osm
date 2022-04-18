package configurator

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
)

const (
	// defaultServiceCertValidityDuration is the default validity duration for service certificates
	defaultServiceCertValidityDuration = 24 * time.Hour

	// defaultCertKeyBitSize is the default certificate key bit size
	defaultCertKeyBitSize = 2048

	// minCertKeyBitSize is the minimum certificate key bit size
	minCertKeyBitSize = 2048

	// maxCertKeyBitSize is the maximum certificate key bit size
	maxCertKeyBitSize = 4096
)

// The functions in this file implement the configurator.Configurator interface

// GetMeshConfig returns the MeshConfig resource corresponding to the control plane
func (c *client) GetMeshConfig() configv1alpha3.MeshConfig {
	return c.getMeshConfig()
}

// GetOSMNamespace returns the namespace in which the OSM controller pod resides.
func (c *client) GetOSMNamespace() string {
	return c.osmNamespace
}

func marshalConfigToJSON(config configv1alpha3.MeshConfigSpec) (string, error) {
	bytes, err := json.MarshalIndent(&config, "", "    ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// GetMeshConfigJSON returns the MeshConfig in pretty JSON.
func (c *client) GetMeshConfigJSON() (string, error) {
	cm, err := marshalConfigToJSON(c.getMeshConfig().Spec)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMeshConfigMarshaling)).Msgf("Error marshaling MeshConfig %s: %+v", c.getMeshConfigCacheKey(), c.getMeshConfig())
		return "", err
	}
	return cm, nil
}

// IsPermissiveTrafficPolicyMode tells us whether the OSM Control Plane is in permissive mode,
// where all existing traffic is allowed to flow as it is,
// or it is in SMI Spec mode, in which only traffic between source/destinations
// referenced in SMI policies is allowed.
func (c *client) IsPermissiveTrafficPolicyMode() bool {
	return c.getMeshConfig().Spec.Traffic.EnablePermissiveTrafficPolicyMode
}

// IsEgressEnabled determines whether egress is globally enabled in the mesh or not.
func (c *client) IsEgressEnabled() bool {
	return c.getMeshConfig().Spec.Traffic.EnableEgress
}

// IsDebugServerEnabled determines whether osm debug HTTP server is enabled
func (c *client) IsDebugServerEnabled() bool {
	return c.getMeshConfig().Spec.Observability.EnableDebugServer
}

// IsTracingEnabled returns whether tracing is enabled
func (c *client) IsTracingEnabled() bool {
	return c.getMeshConfig().Spec.Observability.Tracing.Enable
}

// GetTracingHost is the host to which we send tracing spans
func (c *client) GetTracingHost() string {
	tracingAddress := c.getMeshConfig().Spec.Observability.Tracing.Address
	if tracingAddress != "" {
		return tracingAddress
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", constants.DefaultTracingHost, c.GetOSMNamespace())
}

// GetTracingPort returns the tracing listener port
func (c *client) GetTracingPort() uint32 {
	tracingPort := c.getMeshConfig().Spec.Observability.Tracing.Port
	if tracingPort != 0 {
		return uint32(tracingPort)
	}
	return constants.DefaultTracingPort
}

// GetTracingEndpoint returns the listener's collector endpoint
func (c *client) GetTracingEndpoint() string {
	tracingEndpoint := c.getMeshConfig().Spec.Observability.Tracing.Endpoint
	if tracingEndpoint != "" {
		return tracingEndpoint
	}
	return constants.DefaultTracingEndpoint
}

// GetMaxDataPlaneConnections returns the max data plane connections allowed, 0 if disabled
func (c *client) GetMaxDataPlaneConnections() int {
	return c.getMeshConfig().Spec.Sidecar.MaxDataPlaneConnections
}

// GetEnvoyLogLevel returns the envoy log level
func (c *client) GetEnvoyLogLevel() string {
	logLevel := c.getMeshConfig().Spec.Sidecar.LogLevel
	if logLevel != "" {
		return logLevel
	}
	return constants.DefaultEnvoyLogLevel
}

func (c *client) GetEnvoyProxyMode() configv1alpha3.ProxyMode {
	proxyConfig := c.getMeshConfig().Spec.Sidecar.ProxyMode
	switch proxyConfig {
	case configv1alpha3.ProxyModeLocalhost, configv1alpha3.ProxyModePodIP:
		return proxyConfig
	default:
		// unrecognized proxy mode; return the default: ProxyModeLocalhost
		return configv1alpha3.ProxyModeLocalhost
	}
}

// GetEnvoyImage returns the envoy image
func (c *client) GetEnvoyImage() string {
	image := c.getMeshConfig().Spec.Sidecar.EnvoyImage
	if image == "" {
		image = os.Getenv("OSM_DEFAULT_ENVOY_IMAGE")
	}
	return image
}

// GetEnvoyWindowsImage returns the envoy windows image
func (c *client) GetEnvoyWindowsImage() string {
	image := c.getMeshConfig().Spec.Sidecar.EnvoyWindowsImage
	if image == "" {
		image = os.Getenv("OSM_DEFAULT_ENVOY_WINDOWS_IMAGE")
	}
	return image
}

// GetInitContainerImage returns the init container image
func (c *client) GetInitContainerImage() string {
	image := c.getMeshConfig().Spec.Sidecar.InitContainerImage
	if image == "" {
		image = os.Getenv("OSM_DEFAULT_INIT_CONTAINER_IMAGE")
	}
	return image
}

// GetServiceCertValidityPeriod returns the validity duration for service certificates, and a default in case of invalid duration
func (c *client) GetServiceCertValidityPeriod() time.Duration {
	durationStr := c.getMeshConfig().Spec.Certificate.ServiceCertValidityDuration
	validityDuration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing service certificate validity duration %s", durationStr)
		return defaultServiceCertValidityDuration
	}

	return validityDuration
}

// GetCertKeyBitSize returns the certificate key bit size to be used
func (c *client) GetCertKeyBitSize() int {
	bitSize := c.getMeshConfig().Spec.Certificate.CertKeyBitSize
	if bitSize < minCertKeyBitSize || bitSize > maxCertKeyBitSize {
		log.Error().Msgf("Invalid key bit size: %d", bitSize)
		return defaultCertKeyBitSize
	}

	return bitSize
}

// IsPrivilegedInitContainer returns whether init containers should be privileged
func (c *client) IsPrivilegedInitContainer() bool {
	return c.getMeshConfig().Spec.Sidecar.EnablePrivilegedInitContainer
}

// GetConfigResyncInterval returns the duration for resync interval.
// If error or non-parsable value, returns 0 duration
func (c *client) GetConfigResyncInterval() time.Duration {
	resyncDuration := c.getMeshConfig().Spec.Sidecar.ConfigResyncInterval
	duration, err := time.ParseDuration(resyncDuration)
	if err != nil {
		log.Debug().Err(err).Msgf("Error parsing config resync interval: %s", duration)
		return time.Duration(0)
	}
	return duration
}

// GetProxyResources returns the `Resources` configured for proxies, if any
func (c *client) GetProxyResources() corev1.ResourceRequirements {
	return c.getMeshConfig().Spec.Sidecar.Resources
}

// GetInboundExternalAuthConfig returns the External Authentication configuration for incoming traffic, if any
func (c *client) GetInboundExternalAuthConfig() auth.ExtAuthConfig {
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
func (c *client) GetFeatureFlags() configv1alpha3.FeatureFlags {
	return c.getMeshConfig().Spec.FeatureFlags
}

// GetOSMLogLevel returns the configured OSM log level
func (c *client) GetOSMLogLevel() string {
	return c.getMeshConfig().Spec.Observability.OSMLogLevel
}
