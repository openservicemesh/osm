package configurator

import "errors"

var (
	errSyncingCaches = errors.New("Failed initial cache sync for Configurator informers")
	errInitInformers = errors.New("Configurator informers are not initialized")
)
