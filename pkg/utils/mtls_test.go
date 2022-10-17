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

func TestValidateClient(t *testing.T) {
	assert := tassert.New(t)

	type validateClientTest struct {
		ctx           context.Context
		expectedError error
	}

	certManager := tresorFake.NewFake(1 * time.Hour)
	cnPrefix := fmt.Sprintf("%s.%s.%s", uuid.New(), tests.BookstoreServiceAccountName, tests.Namespace)
	certPEM, _ := certManager.IssueCertificate(certificate.ForCommonNamePrefix(cnPrefix))
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
