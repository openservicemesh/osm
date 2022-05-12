package kube

import "github.com/pkg/errors"

var (
	errParseMulticlusterServiceIP = errors.New("could not parse multicluster service IP")
)
