package kubernetes

import (
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/identity"
)

// GetServiceAccountFromProxyCertificate returns the ServiceAccount information encoded in the certificate CN
func GetServiceAccountFromProxyCertificate(cn certificate.CommonName) (identity.K8sServiceAccount, error) {
	var svcAccount identity.K8sServiceAccount
	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return svcAccount, err
	}

	svcAccount.Name = cnMeta.ServiceAccount
	svcAccount.Namespace = cnMeta.Namespace

	return svcAccount, nil
}
