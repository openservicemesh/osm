package main

import (
	"fmt"
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/vault"
	"github.com/open-service-mesh/osm/pkg/constants"
)

type certificateManagerKind string

// These are the supported certificate issuers.
const (
	// Tresor is an internal package, which leverages Kubernetes secrets and signs certs on the OSM pod
	tresorKind certificateManagerKind = "tresor"

	// Azure Key Vault integration; uses AKV for certificate storage only; certs are signed on the OSM pod
	keyVaultKind = "keyvault"

	// Hashi Vault integration; OSM is pointed to an external Vault; signing of certs happens on Vault
	vaultKind = "vault"
)

// Functions we can call to create a Certificate Manager for each kind of supported certificate issuer
var certManagers = map[certificateManagerKind]func() certificate.Manager{
	tresorKind:   getTresorCertificateManager,
	keyVaultKind: getAzureKeyVaultCertManager,
	vaultKind:    getHashiVaultCertManager,
}

// Get a list of the supported certificate issuers
func getPossibleCertManagers() []string {
	var possible []string
	for kind := range certManagers {
		possible = append(possible, string(kind))
	}
	return possible
}

func getTresorCertificateManager() certificate.Manager {
	rootCert, err := tresor.NewCA(constants.CertificationAuthorityCommonName, getServiceCertValidityPeriod())
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create new Certificate Authority with cert issuer %s", *certManagerKind)
	}

	if rootCert == nil {
		log.Fatal().Msgf("Invalid root certificate created by cert issuer %s", *certManagerKind)
	}

	certManager, err := tresor.NewCertManager(rootCert, getServiceCertValidityPeriod())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to instantiate Azure Key Vault as a Certificate Manager")
	}

	return certManager
}

func getAzureKeyVaultCertManager() certificate.Manager {
	// TODO(draychev): implement: https://github.com/open-service-mesh/osm/issues/577
	log.Fatal().Msg("Azure Key Vault certificate manager is not implemented")
	return nil
}

func getHashiVaultCertManager() certificate.Manager {
	if _, ok := map[string]interface{}{"http": nil, "https": nil}[*vaultProtocol]; !ok {
		log.Fatal().Msgf("Value %s is not a valid Hashi Vault protocol", *vaultProtocol)
	}

	// A Vault address would have the following shape: "http://vault.default.svc.cluster.local:8200"
	vaultAddr := fmt.Sprintf("%s://%s:%d", *vaultProtocol, *vaultHost, *vaultPort)
	vaultCertManager, err := vault.NewCertManager(vaultAddr, *vaultToken, getServiceCertValidityPeriod())
	if err != nil {
		log.Fatal().Err(err).Msg("Error instantiating Hashicorp Vault as a Certificate Manager")
	}

	_, err = vaultCertManager.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootExpiration)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create new Certificate Authority")
	}

	return vaultCertManager
}

func getServiceCertValidityPeriod() time.Duration {
	return time.Duration(validity) * time.Minute
}
