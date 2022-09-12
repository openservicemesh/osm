package kube

import (
	"errors"

	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// kindSvcAccount is the ServiceAccount kind
	kindSvcAccount = "ServiceAccount"
)

var (
	log                = logger.New("kube-provider")
	errServiceNotFound = errors.New("service not found")
)

// client is the type used to represent the k8s client for endpoints and service provider
type client struct {
	// PassthroughInterface is the set of methods that we allow to be exported and used externally, since there are no
	// further abstractions than the k8s client.
	k8s.PassthroughInterface

	kubeController k8s.Controller
}
