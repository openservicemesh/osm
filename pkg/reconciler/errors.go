package reconciler

import "fmt"

var (
	errSyncingCaches = fmt.Errorf("Failed initial cache sync for reconciler informers")
	errInitInformers = fmt.Errorf("Informer not initialized")
)
