package kube

import "github.com/pkg/errors"

var (
	errSyncingCaches   = errors.New("failed initial sync of resources required for ingress")
	errInitInformers   = errors.New("informers are not initialized")
	errServiceNotFound = errors.New("service not found")
	errParseClusterIP  = errors.New("could not parse cluster IP")
)
