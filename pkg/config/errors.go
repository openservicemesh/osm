package config

import "github.com/pkg/errors"

var (
	errSyncingCaches = errors.New("Failed initial cache sync for MultiClusterService informer")
	errInitInformers = errors.New("MultiClusterService informer not initialized")
)
