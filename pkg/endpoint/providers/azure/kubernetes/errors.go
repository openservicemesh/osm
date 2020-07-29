package azure

import "github.com/pkg/errors"

var (
	errSyncingCaches     = errors.New("syncing caches")
	errInvalidObjectType = errors.New("invalid object type in cache")
)
