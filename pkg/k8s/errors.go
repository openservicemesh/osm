package k8s

import "fmt"

var (
	errListingNamespaces = fmt.Errorf("Failed to list monitored namespaces")
	errServiceNotFound   = fmt.Errorf("Service not found")

	// errMoreThanOnePodForUUID is an error for when OSM finds more than one pod for a given xDS certificate. There should always be exactly one Pod for a given xDS certificate.
	errMoreThanOnePodForUUID = fmt.Errorf("found more than one pod for xDS uuid")

	// errDidNotFindPodForUUID is an error for when OSM cannot not find a pod for the given xDS certificate.
	errDidNotFindPodForUUID = fmt.Errorf("did not find pod for uuid")

	// errServiceAccountDoesNotMatchProxy is an error for when the service account of a Pod does not match the xDS certificate.
	errServiceAccountDoesNotMatchProxy = fmt.Errorf("service account does not match proxy")

	// errNamespaceDoesNotMatchProxy is an error for when the namespace of the Pod does not match the xDS certificate.
	errNamespaceDoesNotMatchProxy = fmt.Errorf("namespace does not match proxy")
)
