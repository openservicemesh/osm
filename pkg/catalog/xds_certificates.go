package catalog

import (
	"strings"

	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
)

// NewCertCommonNameWithProxyID returns a newly generated CommonName for a certificate of the form: <ProxyUUID>.<serviceAccount>.<namespace>
func NewCertCommonNameWithProxyID(proxyUUID uuid.UUID, serviceAccount, namespace string) certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{proxyUUID.String(), serviceAccount, namespace}, constants.DomainDelimiter))
}
