package models

import (
	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
)

// TelemetryConfig defines the telemetry configuration applicable to a proxy instance
type TelemetryConfig struct {
	Policy               *policyv1alpha1.Telemetry
	OpenTelemetryService *configv1alpha2.ExtensionService
}
