package kube

import "github.com/pkg/errors"

var (
	errServiceNotFound            = errors.New("service not found")
	errParseClusterIP             = errors.New("could not parse cluster IP")
	errParseMulticlusterServiceIP = errors.New("could not parse multicluster service IP")
)
