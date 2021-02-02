package certificate

import (
	"encoding/pem"
	"io/ioutil"

	"github.com/pkg/errors"

	tresorPem "github.com/openservicemesh/osm/pkg/certificate/pem"
)

// LoadCertificateFromFile loads a certificate from a PEM file.
func LoadCertificateFromFile(caPEMFile string) (tresorPem.Certificate, error) {
	if caPEMFile == "" {
		return nil, errors.Wrap(errInvalidFileName, caPEMFile)
	}

	// #nosec G304
	caPEM, err := ioutil.ReadFile(caPEMFile)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading file: %+v", caPEMFile)
		return nil, err
	}

	caBlock, _ := pem.Decode(caPEM)
	if caBlock == nil || caBlock.Type != TypeCertificate {
		log.Error().Err(err).Msgf("Certificate not found in file: %+v", caPEMFile)
		return nil, errDecodingPEMBlock
	}

	return pem.EncodeToMemory(caBlock), nil
}

// LoadPrivateKeyFromFile loads a private key from a PEM file.
func LoadPrivateKeyFromFile(caKeyPEMFile string) (tresorPem.PrivateKey, error) {
	if caKeyPEMFile == "" {
		log.Error().Msgf("Invalid file for private key: %s", caKeyPEMFile)
		return nil, errInvalidFileName
	}

	// #nosec G304
	caKeyPEM, err := ioutil.ReadFile(caKeyPEMFile)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading file: %+v", caKeyPEMFile)
		return nil, err
	}

	caKeyBlock, _ := pem.Decode(caKeyPEM)
	if caKeyBlock == nil || caKeyBlock.Type != TypePrivateKey {
		log.Error().Err(err).Msgf("Private Key not found in file: %+v", caKeyPEMFile)
		return nil, err
	}

	return pem.EncodeToMemory(caKeyBlock), nil
}
