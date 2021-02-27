package dispatcher

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Dispatcher functions", func() {
	defer GinkgoRecover()
	Context("Test isDeltaUpdate()", func() {
		It("returns false", func() {
			message := PubSubMessage{
				AnnouncementType: EndpointUpdated,
				OldObj:           nil,
				NewObj:           nil,
			}
			actual := isDeltaUpdate(message)
			Expect(actual).To(BeFalse())
		})
	})

	Context("Test flatten()", func() {
		It("flattens the groups of announcement types", func() {
			groups := [][]AnnouncementType{{TCPRouteAdded, TCPRouteDeleted}, {EndpointAdded, EndpointDeleted}}
			expected := []AnnouncementType{TCPRouteAdded, TCPRouteDeleted, EndpointAdded, EndpointDeleted}
			actual := flatten(groups...)
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test Start()", func() {
		It("works", func() {
			stop := make(chan struct{})
			go func() { Start(stop) }()
			close(stop)
		})
	})
})
