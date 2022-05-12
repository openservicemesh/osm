package auth

import (
	"time"
)

// ExtAuthConfig implements a generic subset of External Authz to configure external authorization through HttpFilters
type ExtAuthConfig struct {
	// Enable enables/disables the inbound external authorization policy if present.
	Enable bool

	// Address is the target destination endpoint that will handle external authorization.
	Address string

	// Port is the remote destination port for the external authorization endpoint.
	Port uint16

	// StatPrefix is a prefix for inbound external authorization related metrics.
	StatPrefix string

	// AuthzTimeout defines the timeout to consider for the remote endpoint to reply in time.
	AuthzTimeout time.Duration

	// FailureModeAllow allows specifying if traffic should succeed or fail if the external authorization endpoint fails to respond.
	FailureModeAllow bool
}
