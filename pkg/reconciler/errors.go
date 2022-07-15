package reconciler

import

var (
	errSyncingCaches = errors.New("Failed initial cache sync for reconciler informers")
	errInitInformers = errors.New("Informer not initialized")
)
