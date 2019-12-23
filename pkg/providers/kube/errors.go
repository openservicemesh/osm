package kube

import "errors"

var (
	errNotInCache          = errors.New("TrafficSplit does not exist in cache")
	errRetrievingFromCache = errors.New("retrieving from TrafficSplit cache")
	errBackendNotFound     = errors.New("TrafficSplit backend not found")
	errSyncingCaches       = errors.New("failed initial sync of resources required for ingress")
	errInitInformers       = errors.New("informers are not initialized")
)
