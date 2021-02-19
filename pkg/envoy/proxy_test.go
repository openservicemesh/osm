package envoy

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	svc = "service-name"
	ns  = "some-namespace"
)

var _ = Describe("Test proxy methods", func() {
	certCommonName := certificate.CommonName(fmt.Sprintf("UUID-of-proxy1234566623211353.%s.%s.one.two.three.co.uk", svc, ns))
	certSerialNumber := certificate.SerialNumber("123456")
	podUID := uuid.New().String()
	proxy := NewProxy(certCommonName, certSerialNumber, nil)

	Context("test GetPodUID() with empty Pod Metadata field", func() {
		It("returns correct values", func() {
			Expect(proxy.GetPodUID()).To(Equal(""))
		})
	})

	Context("test correctness proxy object creation", func() {
		It("returns correct values", func() {
			Expect(proxy.GetCertificateCommonName()).To(Equal(certCommonName))
			Expect(proxy.GetCertificateSerialNumber()).To(Equal(certSerialNumber))

			proxy.PodMetadata = &PodMetadata{
				UID: podUID,
			}
			Expect(proxy.GetPodUID()).To(Equal(podUID))
		})
	})
})

func TestStatsHeaders(t *testing.T) {
	const unknown = "unknown"
	tests := []struct {
		name     string
		proxy    Proxy
		expected map[string]string
	}{
		{
			name: "nil metadata",
			proxy: Proxy{
				PodMetadata: nil,
			},
			expected: map[string]string{
				"osm-stats-kind":      unknown,
				"osm-stats-name":      unknown,
				"osm-stats-namespace": unknown,
				"osm-stats-pod":       unknown,
			},
		},
		{
			name: "empty metadata",
			proxy: Proxy{
				PodMetadata: &PodMetadata{},
			},
			expected: map[string]string{
				"osm-stats-kind":      unknown,
				"osm-stats-name":      unknown,
				"osm-stats-namespace": unknown,
				"osm-stats-pod":       unknown,
			},
		},
		{
			name: "full metadata",
			proxy: Proxy{
				PodMetadata: &PodMetadata{
					Name:         "pod",
					Namespace:    "ns",
					WorkloadKind: "kind",
					WorkloadName: "name",
				},
			},
			expected: map[string]string{
				"osm-stats-kind":      "kind",
				"osm-stats-name":      "name",
				"osm-stats-namespace": "ns",
				"osm-stats-pod":       "pod",
			},
		},
		{
			name: "replicaset with expected name format",
			proxy: Proxy{
				PodMetadata: &PodMetadata{
					WorkloadKind: "ReplicaSet",
					WorkloadName: "some-name-randomchars",
				},
			},
			expected: map[string]string{
				"osm-stats-kind":      "Deployment",
				"osm-stats-name":      "some-name",
				"osm-stats-namespace": unknown,
				"osm-stats-pod":       unknown,
			},
		},
		{
			name: "replicaset without expected name format",
			proxy: Proxy{
				PodMetadata: &PodMetadata{
					WorkloadKind: "ReplicaSet",
					WorkloadName: "name",
				},
			},
			expected: map[string]string{
				"osm-stats-kind":      "ReplicaSet",
				"osm-stats-name":      "name",
				"osm-stats-namespace": unknown,
				"osm-stats-pod":       unknown,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.proxy.StatsHeaders()
			assert.Equal(t, test.expected, actual)
		})
	}
}
