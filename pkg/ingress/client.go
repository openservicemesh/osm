package ingress

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// Initialize initializes the client and starts the ingress gateway certificate manager routine
func Initialize(kubeClient kubernetes.Interface, kubeController k8s.Controller, stop chan struct{},
	cfg configurator.Configurator, certProvider certificate.Manager, msgBroker *messaging.Broker) error {
	c := &client{
		kubeClient:     kubeClient,
		kubeController: kubeController,
		cfg:            cfg,
		certProvider:   certProvider,
		msgBroker:      msgBroker,
	}

	if err := c.provisionIngressGatewayCert(stop); err != nil {
		return errors.Wrap(err, "Error provisioning ingress gateway certificate")
	}

	return nil
}
