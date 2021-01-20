package utils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestSetupMutualTLS(t *testing.T) {
	assert := tassert.New(t)

	type setupMutualTLStest struct {
		certPem       []byte
		keyPem        []byte
		ca            []byte
		expectedError string
	}

	certManager := tresor.NewFakeCertManager(nil)
	adsCert, err := certManager.GetRootCertificate()
	assert.Nil(err)

	serverType := "ADS"
	goodCertPem := adsCert.GetCertificateChain()
	goodKeyPem := adsCert.GetPrivateKey()
	goodCA := adsCert.GetIssuingCA()
	emptyByteArray := []byte{}

	setupMutualTLStests := []setupMutualTLStest{
		{emptyByteArray, goodKeyPem, goodCA, "[grpc][mTLS][ADS] Failed loading Certificate ([]) and Key "},
		{goodCertPem, goodKeyPem, emptyByteArray, "[grpc][mTLS][ADS] Failed to append client certs"},
		{goodCertPem, goodKeyPem, goodCA, ""},
	}

	for _, smt := range setupMutualTLStests {
		result, err := setupMutualTLS(true, serverType, smt.certPem, smt.keyPem, smt.ca)
		if err != nil {
			assert.Nil(result)
			assert.Contains(err.Error(), smt.expectedError)
		} else {
			assert.NotNil(result)
			assert.Empty(smt.expectedError)
		}
	}
}

func TestValidateClient(t *testing.T) {
	assert := tassert.New(t)

	type validateClientTest struct {
		ctx           context.Context
		commonNames   map[string]interface{}
		expectedError error
	}

	certManager := tresor.NewFakeCertManager(nil)
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New(), tests.BookstoreServiceAccountName, tests.Namespace))
	certPEM, _ := certManager.IssueCertificate(cn, 1*time.Hour)
	cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())

	goodCommonNameMapping := map[string]interface{}{string(cn): cn}
	badCommonNameMapping := map[string]interface{}{"apple": "pear"}

	validateClientTests := []validateClientTest{
		{context.Background(), nil, status.Error(codes.Unauthenticated, "no peer found")},
		{peer.NewContext(context.TODO(), &peer.Peer{}), nil, status.Error(codes.Unauthenticated, "unexpected peer transport credentials")},
		{peer.NewContext(context.TODO(), &peer.Peer{AuthInfo: credentials.TLSInfo{}}), nil, status.Error(codes.Unauthenticated, "could not verify peer certificate")},
		{peer.NewContext(context.TODO(), &peer.Peer{AuthInfo: tests.NewMockAuthInfo(cert)}), badCommonNameMapping, status.Error(codes.Unauthenticated, "disallowed subject common name")},
		{peer.NewContext(context.TODO(), &peer.Peer{AuthInfo: tests.NewMockAuthInfo(cert)}), goodCommonNameMapping, nil},
	}

	for _, vct := range validateClientTests {
		certCN, certSerialNumber, err := ValidateClient(vct.ctx, vct.commonNames)
		if err != nil {
			assert.Equal(certCN, certificate.CommonName(""))
			assert.Equal(certSerialNumber, certificate.SerialNumber(""))
			assert.True(errors.Is(err, vct.expectedError))
		} else {
			assert.NotNil(certCN)
			assert.NotNil(certSerialNumber)
			assert.Empty(vct.expectedError)
		}
	}
}
