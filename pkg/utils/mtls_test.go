package utils

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
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

	certManager := tresorFake.NewFake(nil, 1*time.Hour)
	adsCert, err := certManager.IssueCertificate("fake-ads", certificate.Internal)

	assert.NoError(err)

	serverType := "ADS"
	goodCertPem := adsCert.GetCertificateChain()
	goodKeyPem := adsCert.GetPrivateKey()
	goodCA := adsCert.GetTrustedCAs()
	var emptyByteArray []byte

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
		expectedError error
	}

	certManager := tresorFake.NewFake(nil, 1*time.Hour)
	cnPrefix := fmt.Sprintf("%s.%s.%s", uuid.New(), tests.BookstoreServiceAccountName, tests.Namespace)
	certPEM, _ := certManager.IssueCertificate(cnPrefix, certificate.Internal)
	cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())

	validateClientTests := []validateClientTest{
		{context.Background(), status.Error(codes.Unauthenticated, "no peer found")},
		{peer.NewContext(context.TODO(), &peer.Peer{}), status.Error(codes.Unauthenticated, "unexpected peer transport credentials")},
		{peer.NewContext(context.TODO(), &peer.Peer{AuthInfo: credentials.TLSInfo{}}), status.Error(codes.Unauthenticated, "could not verify peer certificate")},
		{peer.NewContext(context.TODO(), &peer.Peer{AuthInfo: tests.NewMockAuthInfo(cert)}), nil},
	}

	for _, vct := range validateClientTests {
		certCN, certSerialNumber, err := ValidateClient(vct.ctx)
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
