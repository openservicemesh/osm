package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test patch functions", func() {
	Context("Test updateLabels", func() {
		envoyUID := "abc"
		It("adds", func() {
			pod := tests.NewPodTestFixture("ns", "pod-name")
			pod.Labels = nil
			actual := updateLabels(&pod, envoyUID)
			expected := &JSONPatchOperation{
				Op:    "add",
				Path:  "/metadata/labels",
				Value: map[string]string{"osm-envoy-uid": "abc"},
			}
			Expect(actual).To(Equal(expected))
		})

		It("replaces", func() {
			pod := tests.NewPodTestFixture("ns", "pod-name")
			actual := updateLabels(&pod, envoyUID)
			expected := &JSONPatchOperation{
				Op:    "replace",
				Path:  "/metadata/labels/osm-envoy-uid",
				Value: "abc",
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
