package namespace

import "github.com/pkg/errors"

var (
	errSyncingCaches = errors.New("Failed initial cache sync for Namespace informers")
	errInitInformers = errors.New("Namespace informers are not initialized")
)
