package tresor

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	tresorPem "github.com/open-service-mesh/osm/pkg/tresor/pem"
)

func certFromFile(caPEMFile string) (*x509.Certificate, tresorPem.Certificate, error) {
	if caPEMFile == "" {
		return nil, nil, errors.Wrap(errInvalidFileName, caPEMFile)
	}

	caPEM, err := ioutil.ReadFile(caPEMFile)
	if err != nil {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("failed reading file: %+v", caPEMFile))
	}

	caBlock, _ := pem.Decode(caPEM)
	if caBlock == nil || caBlock.Type != TypeCertificate {
		return nil, nil, errDecodingPEMBlock
	}

	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("failed parsing certificate loaded from %+v", caPEMFile))
	}

	return ca, caBlock.Bytes, nil
}

func privKeyFromFile(caKeyPEMFile string) (*rsa.PrivateKey, tresorPem.PrivateKey, error) {
	if caKeyPEMFile == "" {
		return nil, nil, errors.Wrap(errInvalidFileName, caKeyPEMFile)
	}

	caKeyPEM, err := ioutil.ReadFile(caKeyPEMFile)
	if err != nil {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("faled reading file: %+v", caKeyPEMFile))
	}

	caKeyBlock, _ := pem.Decode(caKeyPEM)
	if caKeyBlock == nil || caKeyBlock.Type != TypePrivateKey {
		return nil, nil, err
	}

	caKeyInterface, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("failed parsing private key loaded from %+v", caKeyPEMFile))
	}

	return caKeyInterface.(*rsa.PrivateKey), caKeyBlock.Bytes, nil
}

// LoadCertificateFromFile loads a certificate from a PEM file.
func LoadCertificateFromFile(caPEMFile string) (tresorPem.Certificate, error) {
	if caPEMFile == "" {
		return nil, errors.Wrap(errInvalidFileName, caPEMFile)
	}

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
