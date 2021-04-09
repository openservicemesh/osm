package ingress

import "github.com/pkg/errors"

var (
	errSyncingCaches = errors.New("Failed initial cache sync for Ingress informer")
	errInitInformers = errors.New("Ingress informer not initialized")

	// ErrUnsupportedAPIVersion indicates the requested version of ingress is unsupported
	ErrUnsupportedAPIVersion = errors.New("Unsupported ingress API version")
)
