package config

import "github.com/pkg/errors"

var (
	errSyncingCaches = errors.New("Failed initial cache sync for RemoteServices informer")
	errInitInformers = errors.New("RemoteServices informer not initialized")
)
