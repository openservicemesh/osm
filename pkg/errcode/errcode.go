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

// String returns the error code as a string, ex. E1000
func (e errCode) String() string {
	return fmt.Sprintf("E%d", e)
}

//nolint: deadcode,varcheck,unused
var errCodeMap = map[errCode]string{
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
}
