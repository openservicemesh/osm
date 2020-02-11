package tresor

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
)

// certFromFile loads a x509 certificate from a PEM file.
func certFromFile(caPEMFile string) (*x509.Certificate, error) {
	if caPEMFile == "" {
		return nil, errors.Wrap(errInvalidFileName, caPEMFile)
	}

	caPEM, err := ioutil.ReadFile(caPEMFile)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed reading file: %+v", caPEMFile))
	}

	caBlock, _ := pem.Decode(caPEM)
	if caBlock == nil || caBlock.Type != TypeCertificate {
		return nil, errDecodingPEMBlock
	}

	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed parsing certificate loaded from %+v", caPEMFile))
	}

	return ca, nil
}

// privKeyFromFile loads a x509 certificate private key from a PEM file.
func privKeyFromFile(caKeyPEMFile string) (*rsa.PrivateKey, error) {
	if caKeyPEMFile == "" {
		return nil, errors.Wrap(errInvalidFileName, caKeyPEMFile)
	}

	caKeyPEM, err := ioutil.ReadFile(caKeyPEMFile)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("faled reading file: %+v", caKeyPEMFile))
	}

	caKeyBlock, _ := pem.Decode(caKeyPEM)
	if caKeyBlock == nil || caKeyBlock.Type != TypePrivateKey {
		return nil, err
	}

	caKeyInterface, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed parsing private key loaded from %+v", caKeyPEMFile))
	}

	return caKeyInterface.(*rsa.PrivateKey), nil
}
