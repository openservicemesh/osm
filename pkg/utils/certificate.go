package utils

import (
	"fmt"
	"strings"

	"github.com/deislabs/smc/pkg/constants"
)

// NewCertCommonNameWithUUID returns a newly generated CommonName for a certificate of the form: <UUID>;<domain>
func NewCertCommonNameWithUUID(domain string) string {
	return fmt.Sprintf("%s%s%s", NewUUIDStr(), constants.CertCommonNameUUIDServiceDelimiter, domain)
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
