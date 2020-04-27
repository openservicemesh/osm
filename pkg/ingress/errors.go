package ingress

import "errors"

var (
	errSyncingCaches            = errors.New("Failed initial cache sync for Ingress informer")
	errInitInformers            = errors.New("Ingress informer not initialized")
	errInvalidObject            = errors.New("Ingress object is invalid")
	errInvalidMonitorAnnotation = errors.New("Ingress resource as an invalid monitor annoation")
)
