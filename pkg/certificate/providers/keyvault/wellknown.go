package keyvault

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/open-service-mesh/osm/pkg/certificate"
)

const (
	suffix = "azure.mesh" // TODO(draychev): why do we need this?
)

func (kv *keyVault) createCertificateWellKnownCA(cn certificate.CommonName, caName string) ([]byte, []byte, error) {
	// TODO(draychev): these certificateNames do not look that good - change them - orr
	certName := kv.getCertificateName(cn)
	log.Trace().Msgf("Create a certificate: %s", certName)
	ctx := context.Background()

	expires := date.UnixTime(time.Now().UTC().Add(kv.validity))
	var privKey []byte
	parameters := keyvault.CertificateCreateParameters{
		CertificatePolicy: &keyvault.CertificatePolicy{
			IssuerParameters: &keyvault.IssuerParameters{
				Name: to.StringPtr(caName), // Unknown
			},
			KeyProperties: &keyvault.KeyProperties{
				Exportable: to.BoolPtr(true),
				KeySize:    to.Int32Ptr(2048),
				KeyType:    keyvault.RSA,
				ReuseKey:   to.BoolPtr(false),
			},
			SecretProperties: &keyvault.SecretProperties{
				ContentType: to.StringPtr("application/x-pkcs12"), // TODO(draychev): is this correct?
			},
			X509CertificateProperties: &keyvault.X509CertificateProperties{
				Subject: to.StringPtr(fmt.Sprintf("CN=%s.%s", certName, suffix)),
			},
		},
		CertificateAttributes: &keyvault.CertificateAttributes{
			Enabled: to.BoolPtr(true),
			Expires: &expires,
		},
		Tags: map[string]*string{
			"OpenServiceMeshID": to.StringPtr(kv.osmID),
		},
	}

	log.Trace().Msg("AAA")
	result, err := kv.client.CreateCertificate(ctx, kv.vaultURL, certName.String(), parameters)
	log.Trace().Msg("BBB")
	if err != nil {
		log.Trace().Msg("CCC")
		log.Fatal().Err(err).Msgf("CreateCertificate error: %v", err)
		return nil, nil, err
	}
	log.Trace().Msg("DDD")
	data, _ := json.Marshal(result)
	log.Trace().Msgf("CreateCertificate: %s", data)
	return data, privKey, nil
}

func (kv *keyVault) importCertificate(b64EncodedCertificate *string, certName string) {
	log.Info()
	ctx := context.Background()

	certToImport := keyvault.CertificateImportParameters{
		Base64EncodedCertificate: b64EncodedCertificate,
	}
	result, err := kv.client.ImportCertificate(ctx, kv.vaultURL, certName, certToImport)
	if err != nil {
		log.Fatal().Msgf("Error Import : %v", err)
	}
	log.Printf("restore result : %v", result)
}

func (kv *keyVault) getCertificateWellKnownCA(certName certName) (cert []byte, privKey []byte, err error) {
	ctx := context.Background()
	certBundle, err := kv.client.GetCertificate(ctx, kv.vaultURL, certName.String(), "")
	if err != nil {
		strErr := err.Error()
		if strings.Contains(strErr, "404") && strings.Contains(strErr, "Certificate not found") {
			return nil, nil, errCertNotFound
		}
		return nil, nil, err
	}
	cert = *certBundle.Cer
	return cert, nil, nil
}
