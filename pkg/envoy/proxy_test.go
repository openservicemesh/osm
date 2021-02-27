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

	Context("test GetLastAppliedVersion()", func() {
		It("returns correct values", func() {
			actual := proxy.GetLastAppliedVersion(TypeCDS)
			Expect(actual).To(Equal(uint64(0)))

			proxy.SetLastAppliedVersion(TypeCDS, uint64(345))

			actual = proxy.GetLastAppliedVersion(TypeCDS)
			Expect(actual).To(Equal(uint64(345)))
		})
	})

	Context("test GetLastSentNonce()", func() {
		It("returns correct values", func() {
			proxy.SetNewNonce(TypeCDS)

			firstNonce := proxy.GetLastSentNonce(TypeCDS)
			Expect(firstNonce).ToNot(Equal(uint64(0)))

			proxy.SetNewNonce(TypeCDS)

			secondNonce := proxy.GetLastSentNonce(TypeCDS)
			Expect(secondNonce).ToNot(Equal(firstNonce))
		})
	})

	Context("test GetLastSentVersion()", func() {
		It("returns correct values", func() {
			actual := proxy.GetLastSentVersion(TypeCDS)
			Expect(actual).To(Equal(uint64(0)))

			newVersion := uint64(132)
			proxy.SetLastSentVersion(TypeCDS, newVersion)

			actual = proxy.GetLastSentVersion(TypeCDS)
			Expect(actual).To(Equal(newVersion))

			proxy.IncrementLastSentVersion(TypeCDS)
			actual = proxy.GetLastSentVersion(TypeCDS)
			Expect(actual).To(Equal(newVersion + 1))
		})
	})

	Context("test GetConnectedAt()", func() {
		It("returns correct values", func() {
			actual := proxy.GetConnectedAt()
			Expect(actual).ToNot(Equal(uint64(0)))
		})
	})

	Context("test HasPodMetadata()", func() {
		It("returns correct values", func() {
			actual := proxy.HasPodMetadata()
			Expect(actual).To(BeFalse())
		})
	})

	Context("test StatsHeaders()", func() {
		It("returns correct values", func() {
			actual := proxy.StatsHeaders()
			expected := map[string]string{
				"osm-stats-namespace": "unknown",
				"osm-stats-kind":      "unknown",
				"osm-stats-name":      "unknown",
				"osm-stats-pod":       "unknown",
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("test correctness proxy object creation", func() {
		It("returns correct values", func() {
			Expect(proxy.GetCertificateCommonName()).To(Equal(certCommonName))
			Expect(proxy.GetCertificateSerialNumber()).To(Equal(certSerialNumber))
			Expect(proxy.HasPodMetadata()).To(BeFalse())

			proxy.PodMetadata = &PodMetadata{
				UID: podUID,
			}

			Expect(proxy.HasPodMetadata()).To(BeTrue())
			Expect(proxy.GetPodUID()).To(Equal(podUID))
			Expect(proxy.String()).To(Equal(fmt.Sprintf("Proxy on Pod with UID=%s", podUID)))
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
