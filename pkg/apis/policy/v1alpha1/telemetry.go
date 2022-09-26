package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Telemetry defines the telemetry configuration for workloads in the mesh.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Telemetry struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the UpstreamTrafficSetting policy specification
	// +optional
	Spec TelemetrySpec `json:"spec,omitempty"`

	// Status is the status of the TelemetryStatus resource.
	// +optional
	Status TelemetryStatus `json:"status,omitempty"`
}

// TelemetrySpec defines the Telemetry specification applicable to workloads
// in the mesh.
type TelemetrySpec struct {
	// Selector defines the pod label selector for pods the Telemetry
	// configuration is applicable to. It selects pods with matching label keys
	// and values. If not specified, the configuration applies to all pods
	// in the Telemetry resource's namespace.
	// +optional
	Selector map[string]string `json:"selector,omitempty"`

	// AccessLog defines the Envoy access log configuration.
	// +optional
	AccessLog *EnvoyAccessLogConfig `json:"accessLog,omitempty"`
}

// EnvoyAccessLogConfig defines the Envoy access log configuration.
type EnvoyAccessLogConfig struct {
	// Format defines the Envoy access log format.
	// The format can either be unstructured or structured (e.g. JSON).
	// Refer to https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log/usage#format-strings
	// regarding how a format string can be specified.
	Format string `json:"format"`

	// OpenTelemetry defines the OpenTelemetry configuration used to export the
	// Envoy access logs to an OpenTelemetry collector.
	// +optional
	OpenTelemetry *EnvoyAccessLogOpenTelemetryConfig `json:"openTelemetry,omitempty"`
}

// EnvoyAccessLogOpenTelemetryConfig defines the Envoy access log OpenTelemetry
// configuration.
type EnvoyAccessLogOpenTelemetryConfig struct {
	// ExtensionService defines the references to ExtensionService resource
	// corresponding to the OpenTelemetry collector.
	ExtensionService ExtensionServiceRef `json:"extensionService"`

	// Attributes defines key-value pairs as additional metadata corresponding access log record.
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
}

// ExtensionServiceRef defines the namespace and name of the ExtensionService resource.
type ExtensionServiceRef struct {
	// Namespace defines the namespaces of the ExtensionService resource.
	Namespace string `json:"namespace"`

	// Name defines the name of the ExtensionService resource.
	Name string `json:"name"`
}

// TelemetryStatus defines the status of a TelemetryStatus resource.
type TelemetryStatus struct {
	// CurrentStatus defines the current status of a TelemetryStatus resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a TelemetryStatus resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// TelemetryList defines the list of TelemetryList objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TelemetryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Telemetry `json:"items"`
}
