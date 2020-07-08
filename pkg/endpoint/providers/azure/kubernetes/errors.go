package azure

import "errors"

var (
	errSyncingCaches     = errors.New("syncing caches")
	errInvalidObjectType = errors.New("invalid object type in cache")
)
