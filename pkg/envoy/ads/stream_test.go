package ads

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
)

func TestIsCNForProxy(t *testing.T) {
	type testCase struct {
		name     string
		cn       certificate.CommonName
		proxy    *envoy.Proxy
		expected bool
	}

	testCases := []testCase{
		{
			name: "workload CN belongs to proxy",
			cn:   certificate.CommonName("svc-acc.namespace.cluster.local"),
			proxy: func() *envoy.Proxy {
				p := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("svc-acc", "namespace"), nil)
				return p
			}(),
			expected: true,
		},
		{
			name: "workload CN does not belong to proxy",
			cn:   certificate.CommonName("svc-acc.namespace.cluster.local"),
			proxy: func() *envoy.Proxy {
				p := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("svc-acc-foo", "namespace"), nil)
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

func findSliceElem(slice []string, elem string) bool {
	for _, v := range slice {
		if v == elem {
			return true
		}
	}
	return false
}

func TestMapsetToSliceConvFunctions(t *testing.T) {
	assert := tassert.New(t)

	discRequest := &xds_discovery.DiscoveryRequest{TypeUrl: "TestTypeurl"}
	discRequest.ResourceNames = []string{"A", "B", "C"}

	nameSet := getRequestedResourceNamesSet(discRequest)

	assert.True(nameSet.Contains("A"))
	assert.True(nameSet.Contains("B"))
	assert.True(nameSet.Contains("C"))
	assert.False(nameSet.Contains("D"))

	nameSlice := getResourceSliceFromMapset(nameSet)

	assert.True(findSliceElem(nameSlice, "A"))
	assert.True(findSliceElem(nameSlice, "B"))
	assert.True(findSliceElem(nameSlice, "C"))
	assert.False(findSliceElem(nameSlice, "D"))
}

var _ = Describe("Test ADS response functions", func() {
	defer GinkgoRecover()
	Context("Test getCertificateCommonNameMeta()", func() {
		It("parses CN into certificateCommonNameMeta", func() {
			proxyUUID := uuid.New()
			testNamespace := uuid.New().String()
			serviceAccount := uuid.New().String()

			cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s.cluster.local", proxyUUID, envoy.KindSidecar, serviceAccount, testNamespace))

			kind, uuid, si, err := getCertificateCommonNameMeta(cn)
			Expect(err).ToNot(HaveOccurred())
			Expect(kind).To(Equal(envoy.KindSidecar))
			Expect(uuid).To(Equal(proxyUUID))
			Expect(si).To(Equal(identity.New(serviceAccount, testNamespace)))
		})

		It("parses CN into certificateCommonNameMeta", func() {
			_, _, _, err := getCertificateCommonNameMeta("a")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test NewXDSCertCommonName() and getCertificateCommonNameMeta() together", func() {
		It("returns the identifier of the form <proxyID>.<kind>.<service-account>.<namespace>", func() {
			proxyUUID := uuid.New()
			serviceAccount := uuid.New().String()
			namespace := uuid.New().String()

			cn := envoy.NewXDSCertCommonName(proxyUUID, envoy.KindSidecar, serviceAccount, namespace)

			actualKind, actualUUID, actualSI, err := getCertificateCommonNameMeta(cn)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualKind).To(Equal(envoy.KindSidecar))
			Expect(actualUUID).To(Equal(proxyUUID))
			Expect(actualSI).To(Equal(identity.New(serviceAccount, namespace)))
		})
	})
})
