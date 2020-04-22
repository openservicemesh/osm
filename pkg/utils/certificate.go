package utils

import (
	"strings"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

const (
	domainDelimiter = "."
)

// NewCertCommonNameWithUUID returns a newly generated CommonName for a certificate of the form: <UUID>.<domain>
func NewCertCommonNameWithUUID(serviceName, namespace, subDomain string) certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{NewUUIDStr(), serviceName, namespace, subDomain}, domainDelimiter))
}

// GetCertificateCommonNameMeta returns the metadata information in the CommonName of a certificate
func GetCertificateCommonNameMeta(cn certificate.CommonName) CertificateCommonNameMeta {
	chunks := strings.Split(cn.String(), domainDelimiter)
	return CertificateCommonNameMeta{
		UUID:        chunks[0],
		ServiceName: chunks[1],
		Namespace:   chunks[2],
		SubDomain:   strings.Join(chunks[3:], domainDelimiter),
	}
}
