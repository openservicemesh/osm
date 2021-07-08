package ads

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

func TestIsCNForProxy(t *testing.T) {
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
			assert := tassert.New(t)

			actual := isCNforProxy(tc.proxy, tc.cn)
			assert.Equal(tc.expected, actual)
		})
	}
}
