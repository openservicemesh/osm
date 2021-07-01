// Package errcode defines the error codes for error messages and an explanation
// of what the error signifies.
package errcode

import (
	"fmt"
)

type errCode int

const (
	// Kind defines the kind for the error code constants
	Kind = "error_code"
)

// Range 1000-1050 is reserved for errors related to application startup or bootstrapping
const (
	// ErrInvalidCLIArgument indicates an invalid CLI argument
	ErrInvalidCLIArgument errCode = iota + 1000

	// ErrSettingLogLevel indicates the specified log level could not be set
	ErrSettingLogLevel

	// ErrParsingMeshConfig indicates the MeshConfig resource could not be parsed
	ErrParsingMeshConfig

	// ErrFetchingControllerPod indicates the osm-controller pod resource could not be fetched
	ErrFetchingControllerPod

	// ErrFetchingInjectorPod indicates the osm-injector pod resource could not be fetched
	ErrFetchingInjectorPod

	// ErrStartingReconcileManager indicates the controller-runtime Manager failed to start
	ErrStartingReconcileManager
)

// Range 2000-2500 is reserved for errors related to traffic policies
const (
	// ErrDedupEgressTrafficMatches indicates an error related to deduplicating egress traffic matches
	ErrDedupEgressTrafficMatches errCode = iota + 2000

	// ErrDedupEgressClusterConfigs indicates an error related to deduplicating egress cluster configs
	ErrDedupEgressClusterConfigs

	// ErrInvalidEgressIPRange indicates the IP address range specified in an egress policy is invalid
	ErrInvalidEgressIPRange

	// ErrInvalidEgressMatches indicates the matches specified in an egress policy is invalid
	ErrInvalidEgressMatches

	// ErrEgressSMIHTTPRouteGroupNotFound indicates the SMI HTTPRouteGroup specified in the egress policy was not found
	ErrEgressSMIHTTPRouteGroupNotFound
)

// String returns the error code as a string, ex. E1000
func (e errCode) String() string {
	return fmt.Sprintf("E%d", e)
}

// Note: error code description mappings must be defined in the same order
// as they appear in the error code definitions - from lowest to highest
// ranges in the order they appear within the range.
//nolint: deadcode,varcheck,unused
var errCodeMap = map[errCode]string{
	//
	// Range 1000-1050
	//
	ErrInvalidCLIArgument: `
An invalid comment line argument was passed to the application.
`,

	ErrSettingLogLevel: `
The specified log level could not be set in the system.
`,

	ErrParsingMeshConfig: `
The 'osm-mesh-config' MeshConfig custom resource could not be parsed.
`,

	ErrFetchingControllerPod: `
The osm-controller k8s pod resource was not able to be retrieved by the system.
`,

	ErrFetchingInjectorPod: `
The osm-injector k8s pod resource was not able to be retrieved by the system.
`,

	ErrStartingReconcileManager: `
The controller-runtime manager to manage the controller used to reconcile the
sidecar injector's MutatingWebhookConfiguration resource failed to start.
`,

	//
	// Range 2000-2500
	//
	ErrDedupEgressTrafficMatches: `
An error was encountered while attempting to deduplicate traffic matching attributes
(destination port, protocol, IP address etc.) used for matching egress traffic.
The applied egress policies could be conflicting with each other, and the system
was unable to process affected egress policies.
`,

	ErrDedupEgressClusterConfigs: `
An error was encountered while attempting to deduplicate upstream clusters associated
with the egress destination.
The applied egress policies could be conflicting with each other, and the system
was unable to process affected egress policies.
`,

	ErrInvalidEgressIPRange: `
An invalid IP address range was specified in the egress policy. The IP address range
must be specified as as a CIDR notation IP address and prefix length, like "192.0.2.0/24",
as defined in RFC 4632.
The invalid IP address range was ignored by the system.
`,

	ErrInvalidEgressMatches: `
An invalid match was specified in the egress policy.
The specified match was ignored by the system while applying the egress policy.
`,

	ErrEgressSMIHTTPRouteGroupNotFound: `
The SMI HTTPRouteGroup resource specified as a match in an egress policy was not found.
Please verify that the specified SMI HTTPRouteGroup resource exists in the same namespace
as the egress policy referencing it as a match.
`,
}
