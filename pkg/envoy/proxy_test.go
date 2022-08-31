package envoy

import (
	"runtime"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test proxy methods", func() {
	proxyUUID := uuid.New()
	proxy := NewProxy(KindSidecar, proxyUUID, identity.New("svc-acc", "namespace"), tests.NewMockAddress("1.2.3.4"), 1)

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

	Context("test UUID", func() {
		It("returns correct values", func() {
			Expect(proxy.UUID).To(Equal(proxyUUID))
		})
	})
})

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
