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
	// Local defines the local rate limiting specification
	// for the upstream host.
	// Local rate limiting is enforced directly by the upstream
	// host without any involvement of a global rate limiting service.
	// This is applied as a token bucket rate limiter.
	// +optional
	Local *LocalRateLimitSpec `json:"local,omitempty"`

	// Global defines the global rate limiting specification
	// for the upstream host.
	// Global rate limiting is enforced by an external rate
	// limiting service.
	// +optional
	Global *GlobalRateLimitSpec `json:"global,omitempty"`
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

// GlobalRateLimitSpec defines the global rate limiting specification
// for the upstream host.
type GlobalRateLimitSpec struct {
	// TCP defines the global rate limiting specification at the network
	// level. This has the ultimate effect of rate limiting connections
	// per unit of time that arrive at the upstream host.
	// +optional
	TCP *TCPGlobalRateLimitSpec `json:"tcp,omitempty"`

	// HTTP defines the global rate limiting specification for HTTP traffic.
	// This has the ultimate effect of rate limiting HTTP requests
	// per unit of time that arrive at the upstream host.
	// +optional
	HTTP *HTTPGlobalRateLimitSpec `json:"http,omitempty"`
}

// TCPGlobalRateLimitSpec defines the global rate limiting specification
// for TCP connections.
type TCPGlobalRateLimitSpec struct {
	// RateLimitService defines the rate limiting service to use
	// as a global rate limiter.
	RateLimitService RateLimitServiceSpec `json:"rateLimitService"`

	// Domain defines a container for a set of rate limits.
	// All domains known to the Ratelimit service must be globally unique.
	// They serve as a way to have different rate limit configurations that
	// don't conflict.
	Domain string `json:"domain"`

	// Descriptors defines the list of rate limit descriptors to use
	// in the rate limit service request.
	Descriptors []TCPRateLimitDescriptor `json:"descriptors"`

	// Timeout defines the timeout interval for calls to the rate limit service.
	// Defaults to 20ms.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// FailureModeDeny defines whether to allow traffic in case of
	// communication failure between rate limiting service and the proxy.
	// Defaults to false.
	// +optional
	FailureModeDeny *bool `json:"failureModeDeny,omitempty"`
}

// RateLimitServiceSpec defines the Rate Limit Service specification.
type RateLimitServiceSpec struct {
	// Host defines the hostname of the rate limiting service.
	Host string `json:"host"`

	// Port defines the port number of the rate limiting service
	Port uint16 `json:"port"`
}

// TCPRateLimitDescriptor defines the rate limit descriptor to use
// in the rate limit service request for TCP connections.
type TCPRateLimitDescriptor struct {
	// Entries defines the list of rate limit descriptor entries.
	Entries []TCPRateLimitDescriptorEntry `json:"entries"`
}

// TCPRateLimitDescriptorEntry defines the rate limit descriptor entry as a
// key-value pair to use in the rate limit service request for TCP connections.
type TCPRateLimitDescriptorEntry struct {
	// Key defines the key of the descriptor entry.
	Key string `json:"key"`

	// Value defines the value of the descriptor entry.
	Value string `json:"value"`
}

// HTTPGlobalRateLimitSpec defines the global rate limiting specification
// for HTTP requests.
type HTTPGlobalRateLimitSpec struct {
	// RateLimitService defines the rate limiting service to use
	// as a global rate limiter.
	RateLimitService RateLimitServiceSpec `json:"rateLimitService"`

	// Domain defines a container for a set of rate limits.
	// All domains known to the Ratelimit service must be globally unique.
	// They serve as a way to have different rate limit configurations that
	// don't conflict.
	Domain string `json:"domain"`

	// Descriptors defines the list of rate limit descriptors to use
	// in the rate limit service request.
	Descriptors []HTTPGlobalRateLimitDescriptor `json:"descriptors"`

	// Timeout defines the timeout interval for calls to the rate limit service.
	// Defaults to 20ms.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// FailureModeDeny defines whether to allow traffic in case of
	// communication failure between rate limiting service and the proxy.
	// Defaults to false.
	// +optional
	FailureModeDeny *bool `json:"failureModeDeny,omitempty"`

	// EnableXRateLimitHeaders defines whether to include the headers
	// X-RateLimit-Limit, X-RateLimit-Remaining, and X-RateLimit-Reset on
	// responses to clients when the rate limit service is consulted for a request.
	// Defaults to false.
	// +optional
	EnableXRateLimitHeaders *bool `json:"enableXRateLimitHeaders,omitempty"`
}

// HTTPGlobalRateLimitDescriptor defines rate limit descriptor to use
// in the rate limit service request for HTTP requests.
type HTTPGlobalRateLimitDescriptor struct {
	// Entries defines the list of rate limit descriptor entries.
	Entries []HTTPGlobalRateLimitDescriptorEntry `json:"entries,omitempty"`
}

// HTTPGlobalRateLimitDescriptorEntry defines the rate limit descriptor entry
// to use in the rate limit service request for HTTP requests.
type HTTPGlobalRateLimitDescriptorEntry struct {
	// GenericKey defines a descriptor entry with a static key-value pair.
	// +optional
	GenericKey *GenericKeyDescriptorEntry `json:"genericKey,omitempty"`

	// RemoteAddress defines a descriptor entry with with key 'remote_address'
	// and value equal to the client's IP address derived from the x-forwarded-for header.
	// +optional
	RemoteAddress *RemoteAddressDescriptorEntry `json:"remoteAddress,omitempty"`

	// RequestHeader defines a descriptor entry that is generated only when the
	// request header matches the given header name. The value of the descriptor
	// entry is derived from the value of the header present in the request.
	// +optional
	RequestHeader *RequestHeaderDescriptorEntry `json:"requestHeader,omitempty"`

	// HeaderValueMatch defines a descriptor entry that is generated when the
	// request header matches the given HTTP header match criteria.
	HeaderValueMatch *HeaderValueMatchDescriptorEntry `json:"headerValueMatch,omitempty"`
}

// GenericKeyDescriptorEntry defines a descriptor entry with a static
// key-value pair.
type GenericKeyDescriptorEntry struct {
	// Value defines the descriptor entry's value.
	Value string `json:"value"`

	// Key defines the descriptor entry's key.
	// Defaults to 'generic_key'.
	// +optional
	Key string `json:"key,omitempty"`
}

// RemoteAddressDescriptorEntry defines a descriptor entry with
// key 'remote_address' and value equal to the client's IP address
// derived from the x-forwarded-for header.
type RemoteAddressDescriptorEntry struct{}

// RequestHeaderDescriptorEntry defines a descriptor entry that is generated only
// when the request header matches the given header name. The value of the descriptor
// entry is derived from the value of the header present in the request.
type RequestHeaderDescriptorEntry struct {
	// Name defines the name of the header used to look up the descriptor entry's value.
	Name string `json:"name"`

	// Key defines the descriptor entry's key.
	Key string `json:"key"`
}

// HeaderValueMatchDescriptorEntry defines the descriptor entry that is generated
// when the request header matches the given HTTP header match criteria.
type HeaderValueMatchDescriptorEntry struct {
	// Value defines the descriptor entry's value.
	Value string `json:"value"`

	// Headers defines the list of HTTP header match criteria.
	Headers []HTTPHeaderMatcher `json:"headers"`

	// Key defines the descriptor entry's key.
	// Defaults to 'header_match'.
	// +optional
	Key string `json:"key,omitempty"`

	// ExpectMatch defines whether the request must match the given
	// match criteria for the descriptor entry to be generated.
	// If set to false, a descriptor entry will be generated when the
	// request does not match the match criteria.
	// Defaults to true.
	// +optional
	ExpectMatch *bool `json:"expectMatch,omitempty"`
}

// HTTPHeaderMatcher defines the HTTP header match criteria.
// Only one of Exact, Prefix, Suffix, Regex, Contains, Present may be set.
type HTTPHeaderMatcher struct {
	// Name defines the name of the header to match.
	Name string `json:"name"`

	// Exact defines the exact value to match.
	// +optional
	Exact string `json:"exact,omitempty"`

	// Prefix defines the prefix value to match.
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// Suffix defines the suffix value to match.
	// +optional
	Suffix string `json:"suffix,omitempty"`

	// Regex defines the regex value to match.
	// +optional
	Regex string `json:"regex,omitempty"`

	// Contains defines the substring value to match.
	// +optional
	Contains string `json:"contains,omitempty"`

	// Present defines whether the request matches the criteria
	// when the header is present. If set to false, header match
	// will be performed based on whether the header is absent.
	Present *bool `json:"present,omitempty"`
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

	// Global defines the global rate limiting specification
	// applied per HTTP route.
	Global *HTTPGlobalPerRouteRateLimitSpec `json:"global,omitempty"`
}

// HTTPGlobalPerRouteRateLimitSpec defines the global rate limiting specification
// applied per HTTP route.
type HTTPGlobalPerRouteRateLimitSpec struct {
	// Descriptors defines the list of rate limit descriptors to use
	// in the rate limit service request.
	Descriptors []HTTPGlobalRateLimitDescriptor `json:"descriptors"`
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
