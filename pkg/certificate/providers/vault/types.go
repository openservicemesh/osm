package vault

import (
	"sync"

	"github.com/hashicorp/vault/api"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
)

// CertManager implements certificate.Manager and contains a Hashi Vault client instance.
type CertManager struct {
	// The Certificate Authority root certificate to be used by this certificate manager
	ca certificate.Certificater

	// The channel announcing to the rest of the system when a certificate has changed
	announcements chan announcements.Announcement

	// Cache for all the certificates issued
	// Types: map[certificate.CommonName]certificate.Certificater
	cache sync.Map

	// Hashicorp Vault client
	client *api.Client

	// The Vault role configured for OSM and passed as a CLI.
	role vaultRole

	cfg configurator.Configurator
}

type vaultRole string

func (vr vaultRole) String() string {
	return string(vr)
}

type vaultPath string

func (vp vaultPath) String() string {
	return string(vp)
}
