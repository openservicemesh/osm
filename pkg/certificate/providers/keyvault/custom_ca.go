package keyvault

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"

	"github.com/open-service-mesh/osm/pkg/certificate"

	"github.com/pkg/errors"

	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
)

const (
	// PEMTypeCertificate is a string constant to be used in the generation of a certificate.
	PEMTypeCertificate = "CERTIFICATE"

	// PEMTypePrivateKey is a string constant to be used in the generation of a private key for a certificate.
	PEMTypePrivateKey = "PRIVATE KEY"

	// MimeTypePEM is mime type for PEM file
	MimeTypePEM = "application/x-pem-file"

	// MimeTypePKCS is the mime type for PKCS
	MimeTypePKCS = "application/x-pkcs12"
)

var errEncodeKey = errors.New("encode key")
var errEncodeCert = errors.New("encode cert")
var errMarshalPrivateKey = errors.New("marshal private key")

// GetSecret retrieves a secret from key vault
func (kv *keyVault) getCertificateCustomCA(certName certName) ([]byte, []byte, error) {
	ctx := context.Background()
	secretBundle, err := kv.client.GetSecret(ctx, kv.vaultURL, certName.String(), "")
	if err != nil {
		strErr := err.Error()
		if strings.Contains(strErr, "404") && strings.Contains(strErr, "Secret not found") {
			return nil, nil, errCertNotFound
		}
		return nil, nil, err
	}

	{
		data, _ := json.Marshal(secretBundle)
		log.Trace().Msgf("Get Secret parsed: %s", data)
		log.Trace().Msgf("Get Secret value: %s", *secretBundle.Value)
	}

	// TODO(draychev): move this elsewhere
	var cert *x509.Certificate
	var privKey *rsa.PrivateKey
	data := []byte(*secretBundle.Value)
	log.Trace().Msgf("Get Secret parsed: %s", data)
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		switch block.Type {
		case PEMTypeCertificate:
			cert, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, err
			}
			log.Trace().Msgf("Certificate: %+v", cert)

		case PEMTypePrivateKey:
			k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, err
			}
			privKey = k.(*rsa.PrivateKey)
			log.Trace().Msgf("Private Key: %+v", privKey)
		}
	}
	if cert == nil || privKey == nil {
		log.Error().Msgf("Error parsing cert and key from Azure Key Vault record for %s", certName)
		return nil, nil, errors.New("invalid secret")

	}
	// TODO(draychev)
	key := []byte{}
	log.Trace().Msgf("Cert=%+v, PrivKey=%+v", cert, key)
	return cert.Raw, key, nil
}

func (kv *keyVault) createCertificateCustomCA(cn certificate.CommonName, caName string) ([]byte, []byte, error) {
	certName := kv.getCertificateName(cn)
	if cert, _, err := kv.getCertificateCustomCA(certName); err == nil {
		log.Info().Msgf("Certificate CN=TODO already exists: %s, %+v", certName, cert)
		// TODO(draychev): compare fingerprints
	}
	// TODO(draychev): these certificateNames do not look that good - change them - orr
	log.Trace().Msgf("Create a certificate: %s", certName)
	ctx := context.Background()

	certPEM, privKeyPEM, expiration, err := tresor.NewCertificate(cn, kv.validity, kv.ca)
	if err != nil {
		return nil, nil, err
	}
	expires := date.UnixTime(*expiration)
	pemBlocks := fmt.Sprintf("%s%s", privKeyPEM, certPEM)

	parameters := keyvault.SecretSetParameters{
		Value: &pemBlocks,
		Tags: map[string]*string{
			"OpenServiceMeshID": to.StringPtr(kv.osmID),
		},
		ContentType: to.StringPtr(MimeTypePEM), // TODO(draychev): is this correct?
		SecretAttributes: &keyvault.SecretAttributes{
			Enabled: to.BoolPtr(true),
			Expires: &expires,
		},
	}

	log.Trace().Msg("AAA")
	result, err := kv.client.SetSecret(ctx, kv.vaultURL, certName.String(), parameters)
	log.Trace().Msg("BBB")
	if err != nil {
		log.Trace().Msg("CCC")
		log.Fatal().Err(err).Msgf("CreateCertificate error: %v", err)
		return nil, nil, err
	}
	log.Trace().Msg("DDD")
	data, _ := json.Marshal(result)
	log.Trace().Msgf("CreateCertificate: %s", data)
	return certPEM, privKeyPEM, nil
}
