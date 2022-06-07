package envoy

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	tassert "github.com/stretchr/testify/assert"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test proxy methods", func() {
	proxyUUID := uuid.New()
	podUID := uuid.New().String()
	proxy := NewProxy(KindSidecar, proxyUUID, identity.New("svc-acc", "namespace"), tests.NewMockAddress("1.2.3.4"))

	It("creates a valid proxy", func() {
		Expect(proxy).ToNot((BeNil()))
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
		It("returns empty if nonce doesn't exist", func() {
			res := proxy.GetLastSentNonce(TypeCDS)
			Expect(res).To(Equal(""))
		})

		It("returns correct values if nonce exists", func() {
			proxy.SetNewNonce(TypeCDS)

			firstNonce := proxy.GetLastSentNonce(TypeCDS)
			Expect(firstNonce).ToNot(Equal(uint64(0)))
			// Platform(Windows): Sleep to accommodate `time.Now()` lower accuracy.
			if runtime.GOOS == constants.OSWindows {
				time.Sleep(1 * time.Millisecond)
			}
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

	Context("test GetIP()", func() {
		It("returns correct values", func() {
			actual := proxy.GetIP()
			Expect(actual.Network()).To(Equal("mockNetwork"))
			Expect(actual.String()).To(Equal("1.2.3.4"))
		})
	})

	Context("test HasPodMetadata()", func() {
		It("returns correct values", func() {
			actual := proxy.HasPodMetadata()
			Expect(actual).To(BeFalse())
		})
	})

	Context("test UUID", func() {
		It("returns correct values", func() {
			Expect(proxy.UUID).To(Equal(proxyUUID))
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
			Expect(proxy.HasPodMetadata()).To(BeFalse())

			proxy.PodMetadata = &PodMetadata{
				UID: podUID,
			}

			Expect(proxy.HasPodMetadata()).To(BeTrue())
			Expect(proxy.PodMetadata.UID).To(Equal(podUID))
			Expect(strings.Contains(proxy.String(), fmt.Sprintf("[ProxyUUID=%s]", proxyUUID))).To(BeTrue())
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

func TestPodMetadataString(t *testing.T) {
	testCases := []struct {
		name     string
		proxy    *Proxy
		expected string
	}{
		{
			name: "with valid pod metadata",
			proxy: &Proxy{
				PodMetadata: &PodMetadata{
					UID:            "some-UID",
					Namespace:      "some-ns",
					Name:           "some-pod",
					ServiceAccount: identity.K8sServiceAccount{Name: "some-service-account"},
				},
			},
			expected: "UID=some-UID, Namespace=some-ns, Name=some-pod, ServiceAccount=some-service-account",
		},
		{
			name: "no pod metadata",
			proxy: &Proxy{
				PodMetadata: nil,
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := tc.proxy.PodMetadataString()
			assert.Equal(tc.expected, actual)
		})
	}
}

func TestSubscribedResources(t *testing.T) {
	assert := tassert.New(t)

	p := Proxy{
		subscribedResources: make(map[TypeURI]mapset.Set),
	}

	res := p.GetSubscribedResources("test")
	assert.Zero(res.Cardinality())

	p.SetSubscribedResources(TypeRDS, mapset.NewSetWith("A", "B", "C"))

	res = p.GetSubscribedResources(TypeRDS)
	assert.Equal(res.Cardinality(), 3)
	assert.True(res.Contains("A"))
	assert.True(res.Contains("B"))
	assert.True(res.Contains("C"))
}
