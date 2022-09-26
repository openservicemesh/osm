package models

import (
	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test proxy methods", func() {
	proxyUUID := uuid.New()
	proxy := NewProxy(KindSidecar, proxyUUID, identity.New("svc-acc", "namespace"), tests.NewMockAddress("1.2.3.4"), 1)

	It("creates a valid proxy", func() {
		Expect(proxy).ToNot((BeNil()))
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
