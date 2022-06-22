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

	// RateLimit specifies the rate limit settings for the traffic
	// directed to the upstream host.
	// If HTTP rate limiting is specified, the rate limiting is applied
	// at the VirtualHost level applicable to all routes within the
	// VirtualHost.
	// +optional
	RateLimit *RateLimitSpec `json:"rateLimit,omitempty"`

	// HTTPRoutes defines the list of HTTP routes settings
	// for the upstream host. Settings are applied at a per
	// route level.
	// +optional
	HTTPRoutes []HTTPRouteSpec `json:"httpRoutes,omitempty"`
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

// RateLimitSpec defines the rate limiting specification for
// the upstream host.
type RateLimitSpec struct {
	// Local specified the local rate limiting specification
	// for the upstream host.
	// Local rate limiting is enforced directly by the upstream
	// host without any involvement of a global rate limiting service.
	// This is applied as a token bucket rate limiter.
	// +optional
	Local *LocalRateLimitSpec `json:"local,omitempty"`
}

// LocalRateLimitSpec defines the local rate limiting specification
// for the upstream host.
type LocalRateLimitSpec struct {
	// TCP defines the local rate limiting specification at the network
	// level. This is a token bucket rate limiter where each connection
	// consumes a single token. If the token is available, the connection
	// will be allowed. If no tokens are available, the connection will be
	// immediately closed.
	// +optional
	TCP *TCPLocalRateLimitSpec `json:"tcp,omitempty"`

	// HTTP defines the local rate limiting specification for HTTP traffic.
	// This is a token bucket rate limiter where each request consumes
	// a single token. If the token is available, the request will be
	// allowed. If no tokens are available, the request will receive the
	// configured rate limit status.
	HTTP *HTTPLocalRateLimitSpec `json:"http,omitempty"`
}

// TCPLocalRateLimitSpec defines the local rate limiting specification
// for the upstream host at the TCP level.
type TCPLocalRateLimitSpec struct {
	// Connections defines the number of connections allowed
	// per unit of time before rate limiting occurs.
	Connections uint32 `json:"connections"`

	// Unit defines the period of time within which connections
	// over the limit will be rate limited.
	// Valid values are "second", "minute" and "hour".
	Unit string `json:"unit"`

	// Burst defines the number of connections above the baseline
	// rate that are allowed in a short period of time.
	// +optional
	Burst uint32 `json:"burst,omitempty"`
}

// HTTPLocalRateLimitSpec defines the local rate limiting specification
// for the upstream host at the HTTP level.
type HTTPLocalRateLimitSpec struct {
	// Requests defines the number of requests allowed
	// per unit of time before rate limiting occurs.
	Requests uint32 `json:"requests"`

	// Unit defines the period of time within which requests
	// over the limit will be rate limited.
	// Valid values are "second", "minute" and "hour".
	Unit string `json:"unit"`

	// Burst defines the number of requests above the baseline
	// rate that are allowed in a short period of time.
	// +optional
	Burst uint32 `json:"burst,omitempty"`

	// ResponseStatusCode defines the HTTP status code to use for responses
	// to rate limited requests. Code must be in the 400-599 (inclusive)
	// error range. If not specified, a default of 429 (Too Many Requests) is used.
	// See https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/v3/http_status.proto#enum-type-v3-statuscode
	// for the list of HTTP status codes supported by Envoy.
	// +optional
	ResponseStatusCode uint32 `json:"responseStatusCode,omitempty"`

	// ResponseHeadersToAdd defines the list of HTTP headers that should be
	// added to each response for requests that have been rate limited.
	// +optional
	ResponseHeadersToAdd []HTTPHeaderValue `json:"responseHeadersToAdd,omitempty"`
}

// HTTPHeaderValue defines an HTTP header name/value pair
type HTTPHeaderValue struct {
	// Name defines the name of the HTTP header.
	Name string `json:"name"`

	// Value defines the value of the header corresponding to the name key.
	Value string `json:"value"`
}

// HTTPRouteSpec defines the settings correspondng to an HTTP route
type HTTPRouteSpec struct {
	// Path defines the HTTP path.
	Path string `json:"path"`

	// RateLimit defines the HTTP rate limiting specification for
	// the specified HTTP route.
	RateLimit *HTTPPerRouteRateLimitSpec `json:"rateLimit,omitempty"`
}

// HTTPPerRouteRateLimitSpec defines the rate limiting specification
// per HTTP route.
type HTTPPerRouteRateLimitSpec struct {
	// Local defines the local rate limiting specification
	// applied per HTTP route.
	Local *HTTPLocalRateLimitSpec `json:"local,omitempty"`
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
