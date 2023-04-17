package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	// defaultServiceCertValidityDuration is the default validity duration for service certificates
	defaultServiceCertValidityDuration = 24 * time.Hour

	// defaultIngressGatewayCertValidityDuration is the default validity duration for ingress gateway certificates
	defaultIngressGatewayCertValidityDuration = 24 * time.Hour

	// defaultCertKeyBitSize is the default certificate key bit size
	defaultCertKeyBitSize = 2048

	// minCertKeyBitSize is the minimum certificate key bit size
	minCertKeyBitSize = 2048

	// maxCertKeyBitSize is the maximum certificate key bit size
	maxCertKeyBitSize = 4096
)

// MeshConfigToJSON returns the MeshConfig in pretty JSON.
func MeshConfigToJSON(mc v1alpha2.MeshConfig) (string, error) {
	bytes, err := json.MarshalIndent(&mc.Spec, "", "    ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// GetTracingHost is the host to which we send tracing spans
func GetTracingHost(mc v1alpha2.MeshConfig) string {
	tracingAddress := mc.Spec.Observability.Tracing.Address
	if tracingAddress != "" {
		return tracingAddress
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", constants.DefaultTracingHost, mc.Namespace)
}

// GetTracingPort returns the tracing listener port
func GetTracingPort(mc v1alpha2.MeshConfig) uint32 {
	tracingPort := mc.Spec.Observability.Tracing.Port
	if tracingPort != 0 {
		return uint32(tracingPort)
	}
	return constants.DefaultTracingPort
}

// GetTracingEndpoint returns the listener's collector endpoint
func GetTracingEndpoint(mc v1alpha2.MeshConfig) string {
	tracingEndpoint := mc.Spec.Observability.Tracing.Endpoint
	if tracingEndpoint != "" {
		return tracingEndpoint
	}
	return constants.DefaultTracingEndpoint
}

// GetEnvoyImage returns the envoy image
func GetEnvoyImage(mc v1alpha2.MeshConfig) string {
	image := mc.Spec.Sidecar.EnvoyImage
	if image == "" {
		image = os.Getenv("OSM_DEFAULT_ENVOY_IMAGE")
	}
	return image
}

// GetEnvoyWindowsImage returns the envoy windows image
func GetEnvoyWindowsImage(mc v1alpha2.MeshConfig) string {
	image := mc.Spec.Sidecar.EnvoyWindowsImage
	if image == "" {
		image = os.Getenv("OSM_DEFAULT_ENVOY_WINDOWS_IMAGE")
	}
	return image
}

// GetInitContainerImage returns the init container image
func GetInitContainerImage(mc v1alpha2.MeshConfig) string {
	image := mc.Spec.Sidecar.InitContainerImage
	if image == "" {
		image = os.Getenv("OSM_DEFAULT_INIT_CONTAINER_IMAGE")
	}
	return image
}

// GetServiceCertValidityPeriod returns the validity duration for service certificates, and a default in case of invalid duration
func GetServiceCertValidityPeriod(mc v1alpha2.MeshConfig) time.Duration {
	durationStr := mc.Spec.Certificate.ServiceCertValidityDuration
	validityDuration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing service certificate validity duration %s", durationStr)
		return defaultServiceCertValidityDuration
	}

	return checkValidityDuration(validityDuration)
}

// GetIngressGatewayCertValidityPeriod returns the validity duration for ingress gateway certificates, and a default in case of unspecified or invalid duration
func GetIngressGatewayCertValidityPeriod(mc v1alpha2.MeshConfig) time.Duration {
	ingressGatewayCertSpec := mc.Spec.Certificate.IngressGateway
	if ingressGatewayCertSpec == nil {
		log.Warn().Msgf("Attempting to get the ingress gateway certificate validity duration even though a cert has not been specified in the mesh config")
		return defaultIngressGatewayCertValidityDuration
	}
	validityDuration, err := time.ParseDuration(ingressGatewayCertSpec.ValidityDuration)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing ingress gateway certificate validity duration %s", ingressGatewayCertSpec.ValidityDuration)
		return defaultIngressGatewayCertValidityDuration
	}

	return checkValidityDuration(validityDuration)
}

func checkValidityDuration(validityDuration time.Duration) time.Duration {
	renewalPeriod := time.Duration(2*certificate.MinRotateBeforeExpireMinutes) * time.Minute
	if validityDuration < renewalPeriod {
		validityDuration = renewalPeriod
		log.Warn().Msgf("Minimum accepted validity duration must be 2x the renewal period - setting validity duration to %v", validityDuration)
	}
	return validityDuration
}

// GetCertKeyBitSize returns the certificate key bit size to be used
func GetCertKeyBitSize(mc v1alpha2.MeshConfig) int {
	bitSize := mc.Spec.Certificate.CertKeyBitSize
	if bitSize < minCertKeyBitSize || bitSize > maxCertKeyBitSize {
		log.Error().Msgf("Invalid key bit size: %d", bitSize)
		return defaultCertKeyBitSize
	}

	return bitSize
}

// ExternalAuthConfigFromMeshConfig returns the External Authentication configuration for incoming traffic, if any
func ExternalAuthConfigFromMeshConfig(mc v1alpha2.MeshConfig) auth.ExtAuthConfig {
	extAuthConfig := auth.ExtAuthConfig{}
	inboundExtAuthzMeshConfig := mc.Spec.Traffic.InboundExternalAuthorization

	extAuthConfig.Enable = inboundExtAuthzMeshConfig.Enable
	extAuthConfig.Address = inboundExtAuthzMeshConfig.Address
	extAuthConfig.Port = uint16(inboundExtAuthzMeshConfig.Port)
	extAuthConfig.StatPrefix = inboundExtAuthzMeshConfig.StatPrefix
	extAuthConfig.FailureModeAllow = inboundExtAuthzMeshConfig.FailureModeAllow
	extAuthConfig.InitialMetadata = inboundExtAuthzMeshConfig.InitialMetadata

	duration, err := time.ParseDuration(inboundExtAuthzMeshConfig.Timeout)
	if err != nil {
		log.Debug().Err(err).Msgf("ExternAuthzTimeout: Not a valid duration %s. defaulting to 1s.", duration)
		duration = 1 * time.Second
	}
	extAuthConfig.AuthzTimeout = duration

	return extAuthConfig
}
