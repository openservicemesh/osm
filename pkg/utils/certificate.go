package utils

import (
	"fmt"
	"strings"

	"github.com/open-service-mesh/osm/pkg/constants"
)

const (
	domainDelimiter = "."
)

// NewCertCommonNameWithUUID returns a newly generated CommonName for a certificate of the form: <UUID>;<domain>
func NewCertCommonNameWithUUID(serviceAccountName, namespace, subDomain string) string {
	return fmt.Sprintf("%s%s%s.%s.%s", NewUUIDStr(), constants.CertCommonNameUUIDServiceDelimiter, serviceAccountName, namespace, subDomain)
}

// GetCertificateCommonNameMeta returns the metadata information in the CommonName of a certificate
func GetCertificateCommonNameMeta(cn string) CertificateCommonNameMeta {
	var cnMeta CertificateCommonNameMeta
	var cnWithUUIDStripped string

	if strings.Contains(cn, constants.CertCommonNameUUIDServiceDelimiter) {
		tmp := strings.Split(cn, constants.CertCommonNameUUIDServiceDelimiter)
		cnMeta.UUID = tmp[0]
		cnWithUUIDStripped = tmp[1]
	} else {
		cnWithUUIDStripped = cn
	}

	maxSplits := 3 // <service_account_name>.<namespace>.<sub_domain>
	tmp := strings.SplitN(cnWithUUIDStripped, domainDelimiter, maxSplits)
	if len(tmp) != maxSplits {
		panic("Certificate Common Name domain should be of the form <service_account_name>.<namespace>.<sub_domain>")
	}

	cnMeta.ServiceAccountName = tmp[0]
	cnMeta.Namespace = tmp[1]
	cnMeta.SubDomain = tmp[2]

	return cnMeta

}

// GetCertCommonNameWithoutUUID returns the CommonName with the UUID stripped if the UUID delimiter exists, otherwise returns the existing name
func GetCertCommonNameWithoutUUID(cn string) string {
	if strings.Contains(cn, constants.CertCommonNameUUIDServiceDelimiter) {
		tmp := strings.Split(cn, constants.CertCommonNameUUIDServiceDelimiter)
		cnWithUUIDStripped := tmp[1]
		return cnWithUUIDStripped
	}

	return cn
}
