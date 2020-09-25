package utils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestSetupMutualTLS(t *testing.T) {
	assert := assert.New(t)

	type setupMutualTLStest struct {
		certPem []byte
		keyPem  []byte
		ca      []byte
	}

	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	adsCert, err := certManager.GetRootCertificate()
	assert.Nil(err)

	serverType := "ADS"
	goodCertPem := adsCert.GetCertificateChain()
	goodKeyPem := adsCert.GetPrivateKey()
	goodCA := adsCert.GetIssuingCA()
	emptyByteArray := []byte{}

	setupMutualTLStests := []setupMutualTLStest{
		{emptyByteArray, goodKeyPem, goodCA},
		{goodCertPem, goodKeyPem, emptyByteArray},
		{goodCertPem, goodKeyPem, goodCA},
	}

	for _, smt := range setupMutualTLStests {
		result, err := setupMutualTLS(true, serverType, smt.certPem, smt.keyPem, smt.ca)
		if err != nil {
			assert.Nil(result)
			assert.NotNil(err)
		} else {
			assert.NotNil(result)
			assert.Nil(err)
		}

	}
}

func TestValidateClient(t *testing.T) {
	assert := assert.New(t)

	type validateClientTest struct {
		ctx           context.Context
		commonNames   map[string]interface{}
		expectedError error
	}

	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New(), tests.BookstoreServiceAccountName, tests.Namespace))
	certPEM, _ := certManager.IssueCertificate(cn, nil)
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

		result, err := ValidateClient(vct.ctx, vct.commonNames)
		if err != nil {
			assert.Equal(result, certificate.CommonName(""))
			assert.True(errors.Is(err, vct.expectedError))

		} else {
			assert.NotNil(result)
			assert.Nil(err)
		}
	}

}
