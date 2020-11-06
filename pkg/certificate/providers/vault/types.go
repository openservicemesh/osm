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
	cache     *map[certificate.CommonName]certificate.Certificater
	cacheLock sync.Mutex

	// Hashicorp Vault client
	client *api.Client

	// The Vault role configured for OSM and passed as a CLI.
	vaultRole string

	cfg configurator.Configurator
}
