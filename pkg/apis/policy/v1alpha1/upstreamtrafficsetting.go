package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpstreamTrafficSetting defines the settings applicable to traffic destined
// to an upstream host.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UpstreamTrafficSetting struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the UpstreamTrafficSetting policy specification
	// +optional
	Spec UpstreamTrafficSettingSpec `json:"spec,omitempty"`

	// Status is the status of the UpstreamTrafficSetting resource.
	// +optional
	Status UpstreamTrafficSettingStatus `json:"status,omitempty"`
}

// UpstreamTrafficSettingSpec defines the upstream traffic setting specification.
type UpstreamTrafficSettingSpec struct {
	// Host the upstream traffic is directed to.
	// Must either be an FQDN corresponding to the upstream service
	// or the name of the upstream service. If only the service name
	// is specified, the FQDN is derived from the service name and
	// the namespace of the UpstreamTrafficSetting rule.
	Host string `json:"host"`

	// ConnectionSettings specifies the connection settings for traffic
	// directed to the upstream host.
	// +optional
	ConnectionSettings *ConnectionSettingsSpec `json:"connectionSettings,omitempty"`
}

// ConnectionSettingsSpec defines the connection settings for an
// upstream host.
type ConnectionSettingsSpec struct {
	// TCP specifies the TCP level connection settings.
	// Applies to both TCP and HTTP connections.
	// +optional
	TCP *TCPConnectionSettings `json:"tcp,omitempty"`

	// HTTP specifies the HTTP level connection settings.
	// +optional
	HTTP *HTTPConnectionSettings `json:"http,omitempty"`
}

// TCPConnectionSettings defines the TCP connection settings for an
// upstream host.
type TCPConnectionSettings struct {
	// MaxConnections specifies the maximum number of TCP connections
	// allowed to the upstream host.
	// Defaults to 4294967295 (2^32 - 1) if not specified.
	// +optional
	MaxConnections *uint32 `json:"maxConnections,omitempty"`

	// ConnectTimeout specifies the TCP connection timeout.
	// Defaults to 5s if not specified.
	// +optional
	ConnectTimeout *metav1.Duration `json:"connectTimeout,omitempty"`
}

// HTTPConnectionSettings defines the HTTP connection settings for an
// upstream host.
type HTTPConnectionSettings struct {
	// MaxRequests specifies the maximum number of parallel requests
	// allowed to the upstream host.
	// Defaults to 4294967295 (2^32 - 1) if not specified.
	// +optional
	MaxRequests *uint32 `json:"maxRequests,omitempty"`

	// MaxRequestsPerConnection specifies the maximum number of requests
	// per connection allowed to the upstream host.
	// Defaults to unlimited if not specified.
	// +optional
	MaxRequestsPerConnection *uint32 `json:"maxRequestsPerConnection,omitempty"`

	// MaxPendingRequests specifies the maximum number of pending HTTP
	// requests allowed to the upstream host. For HTTP/2 connections,
	// if `maxRequestsPerConnection` is not configured, all requests will
	// be multiplexed over the same connection so this circuit breaker
	// will only be hit when no connection is already established.
	// Defaults to 4294967295 (2^32 - 1) if not specified.
	// +optional
	MaxPendingRequests *uint32 `json:"maxPendingRequests,omitempty"`

	// MaxRetries specifies the maximum number of parallel retries
	// allowed to the upstream host.
	// Defaults to 4294967295 (2^32 - 1) if not specified.
	// +optional
	MaxRetries *uint32 `json:"maxRetries,omitempty"`
}

// UpstreamTrafficSettingStatus defines the status of an UpstreamTrafficSetting resource.
type UpstreamTrafficSettingStatus struct {
	// CurrentStatus defines the current status of an UpstreamTrafficSetting resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of an UpstreamTrafficSetting resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// UpstreamTrafficSettingList defines the list of UpstreamTrafficSetting objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UpstreamTrafficSettingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []UpstreamTrafficSetting `json:"items"`
}
