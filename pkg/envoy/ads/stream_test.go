package ads

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
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestIsCNForProxy(t *testing.T) {
	assert := tassert.New(t)

	type testCase struct {
		name     string
		cn       certificate.CommonName
		proxy    *envoy.Proxy
		expected bool
	}

	certSerialNumber := certificate.SerialNumber("123456")

	testCases := []testCase{
		{
			name: "workload CN belongs to proxy",
			cn:   certificate.CommonName("svc-acc.namespace.cluster.local"),
			proxy: func() *envoy.Proxy {
				p, _ := envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.svc-acc.namespace", uuid.New(), envoy.KindSidecar)), certSerialNumber, nil)
				return p
			}(),
			expected: true,
		},
		{
			name: "workload CN does not belong to proxy",
			cn:   certificate.CommonName("svc-acc.namespace.cluster.local"),
			proxy: func() *envoy.Proxy {
				p, _ := envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.svc-acc-foo.namespace", uuid.New(), envoy.KindSidecar)), certSerialNumber, nil)
				return p
			}(),
			expected: false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			actual := isCNforProxy(tc.proxy, tc.cn)
			assert.Equal(tc.expected, actual)
		})
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
