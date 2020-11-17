package kubernetes

import "github.com/pkg/errors"

var (
	errSyncingCaches     = errors.New("Failed initial cache sync for Namespace informers")
	errInitInformers     = errors.New("Informer not initialized")
	errListingNamespaces = errors.New("Failed to list monitored namespaces")
	errServiceNotFound   = errors.New("Service not found")
)
