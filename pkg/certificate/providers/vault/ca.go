package vault

import (
	"time"

	"github.com/hashicorp/vault/api"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor/pem"
)

const (
	vaultRole = "open-service-mesh"
	maxTTL    = 3 * time.Minute
)

// NewCA creates a new certification authority within Hashi Vault. Returns the new root certificate WITHOUT the private key.
func (cm *CertManager) NewCA(cn certificate.CommonName, validity time.Duration) (certificate.Certificater, error) {
	if _, err := cm.client.Logical().Write("pki/root/generate/internal", getIssuanceData(cn, validity)); err != nil {
		log.Error().Err(err).Msg("Error creating a new root certificate")
		return nil, err
	}

	// TODO(draychev): Issue and use an intermediate certificated with "pki_int/issue/" (https://github.com/open-service-mesh/osm/issues/507)

	options := map[string]interface{}{
		"allow_any_name":    "true",
		"allow_subdomains":  "true",
		"allow_baredomains": "true",
		"allow_localhost":   "true",
		"max_ttl":           getDurationInMinutes(maxTTL),
	}

	if _, err := cm.client.Logical().Write(getRoleConfigURL(), options); err != nil {
		return nil, err
	}

	// Ensure cert generation has been initialized correctly
	secret, err := cm.client.Logical().Write(getIssueURL(), getIssuanceData("localhost", cm.validity))
	if err != nil {
		log.Error().Err(err).Msg("Error creating a test certificate with the newly instantiated Hashi Vault client")
		return nil, err
	}

	return newRootCert(cn, secret, time.Now().Add(cm.validity)), nil
}

func newRootCert(cn certificate.CommonName, secret *api.Secret, expiration time.Time) *Certificate {
	return &Certificate{
		commonName: cn,
		expiration: expiration,
		certChain:  pem.Certificate(secret.Data[issuingCAField].(string)),
		issuingCA:  pem.RootCertificate(secret.Data[issuingCAField].(string)),
	}
}
