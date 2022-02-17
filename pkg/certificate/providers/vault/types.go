// Package vault implements the certificate.Manager interface for Hashicorp Vault as the certificate provider.
package vault

import (
	"sync"
	"time"

	"github.com/hashicorp/vault/api"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// CertManager implements certificate.Manager and contains a Hashi Vault client instance.
type CertManager struct {
	// The Certificate Authority root certificate to be used by this certificate manager
	ca *certificate.Certificate

	// Cache for all the certificates issued
	// Types: map[certificate.CommonName]*certificate.Certificate
	cache sync.Map

	// Hashicorp Vault client
	client *api.Client

	// The Vault role configured for OSM and passed as a CLI.
	role vaultRole

	cfg configurator.Configurator

	serviceCertValidityDuration time.Duration

	msgBroker *messaging.Broker
}

type vaultRole string

func (vr vaultRole) String() string {
	return string(vr)
}

type vaultPath string

func (vp vaultPath) String() string {
	return string(vp)
}
