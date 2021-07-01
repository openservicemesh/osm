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

	// ErrFetchingSMIHTTPRouteGroupForTrafficTarget indicates the SMI HTTPRouteGroup specified as a match in an SMI
	// TrafficTarget resource was not able to be retrieved
	ErrFetchingSMIHTTPRouteGroupForTrafficTarget

	// ErrSMIHTTPRouteGroupNoMatch indicates the SMI HTTPRouteGroup resource has no matches specified
	ErrSMIHTTPRouteGroupNoMatch

	// ErrMultipleSMISplitPerServiceUnsupported indicates multiple SMI split policies per service exists and is not supported
	ErrMultipleSMISplitPerServiceUnsupported

	// ErrAddingRouteToOutboundTrafficPolicy indicates there was an error adding a route to an outbound traffic policy
	ErrAddingRouteToOutboundTrafficPolicy

	// ErrFetchingServiceForTrafficTargetDestination indicates an error retrieving services associated with a TrafficTarget destination
	ErrFetchingServiceForTrafficTargetDestination
)

// Range 3000-3500 is reserved for errors related to k8s constructs (service accounts, namespaces, etc.)
const (
	// ErrServiceHostnames indicates the hostnames associated with a service could not be computed
	ErrServiceHostnames errCode = iota + 3000

	// ErrNoMatchingServiceForServiceAccount indicates there are no services associated with the service account
	ErrNoMatchingServiceForServiceAccount
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

	ErrFetchingSMIHTTPRouteGroupForTrafficTarget: `
The SMI HTTPRouteGroup resources specified as a match in an SMI TrafficTarget policy was
unable to be retrieved by the system.
The associated SMI TrafficTarget policy was ignored by the system. Please verify that the
matches specified for the Traffictarget resource exist in the same namespace as the
TrafficTarget policy referencing the match.
`,

	ErrSMIHTTPRouteGroupNoMatch: `
The SMI HTTPRouteGroup resource is invalid as it does not have any matches specified.
The SMI HTTPRouteGroup policy was ignored by the system.
`,

	ErrMultipleSMISplitPerServiceUnsupported: `
There are multiple SMI traffic split policies associated with the same apex(root)
service specified in the policies. The system does not support this scenario so
onlt the first encountered policy is processed by the system, subsequent policies
referring the same apex service are ignored.
`,

	ErrAddingRouteToOutboundTrafficPolicy: `
There was an error adding a route match to an outbound traffic policy representation
within the system.
The associated route was ignored by the system.
`,

	ErrFetchingServiceForTrafficTargetDestination: `
The system was unable to lookup the services associated with the destination specified
in the SMI TrafficTarget policy.
The associated SMI TrafficTarget policy was ignored by the system.
`,

	//
	// Range 3000-3500
	//
	ErrServiceHostnames: `
The hostnames (FQDN) used to access the k8s service could not be retrieved by the system.
Any HTTP traffic policy associated with this service was ignored by the system.
`,

	ErrNoMatchingServiceForServiceAccount: `
The system expected k8s services to be associated with a service account, but no such
service was found. A service matches a service account if the pods backing the service
belong to the service account.
`,
}
