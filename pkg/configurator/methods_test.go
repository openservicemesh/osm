package configurator

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testclient "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Test Envoy configuration creation", func() {
	Context("create envoy config", func() {
		kubeClient := testclient.NewSimpleClientset()
		cfg := NewFakeConfigurator([]string{"foo", "bar"}, kubeClient)
		It("correctly identifies foo as monitored namespace", func() {
			actual := cfg.IsMonitoredNamespace("foo")
			Expect(actual).To(BeTrue())
		})
		It("correctly identifies baz as non monitored namespace", func() {
			actual := cfg.IsMonitoredNamespace("baz")
			Expect(actual).To(BeFalse())
		})
	})
})
