package smi

import "github.com/pkg/errors"

var (
	errSyncingCaches = errors.New("failed initial sync of resources required for ingress")
	errInitInformers = errors.New("informers are not initialized")
)
