// Package models implements the HealthProbe struct.
package models

import "time"

// HealthProbe represents a health probe.
type HealthProbe struct {
	Path    string
	Port    int32
	Timeout time.Duration

	// isHTTP corresponds to an httpGet probe with a scheme of HTTP or undefined.
	// This helps inform what kind of Envoy config to add to the pod. A HealthProbe
	// that is neither HTTP nor TCPSocket is assumed to be HTTPS
	IsHTTP bool

	// isTCPSocket indicates if the probe defines a TCPSocketAction.
	IsTCPSocket bool
}

// HealthProbes is to serve as an indication of whether the given healthProbe has been rewritten
type HealthProbes struct {
	Liveness, Readiness, Startup *HealthProbe
}

// UsesTCP returns true if any of the configured probes uses a TCP probe.
func (probes *HealthProbes) UsesTCP() bool {
	return (probes.Liveness != nil && probes.Liveness.IsTCPSocket) ||
		(probes.Readiness != nil && probes.Readiness.IsTCPSocket) ||
		(probes.Startup != nil && probes.Startup.IsTCPSocket)
}
