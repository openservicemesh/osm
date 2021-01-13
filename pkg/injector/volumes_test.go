package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/featureflags"
)

var _ = Describe("Test volume functions", func() {
	Context("Test getVolumeSpec", func() {
		It("creates volume spec", func() {
			actual := getVolumeSpec("-envoy-config-")
			expected := []v1.Volume{{
				Name: "envoy-bootstrap-config-volume",
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "-envoy-config-",
					},
				},
			}}
			Expect(actual).To(Equal(expected))
		})

		It("creates a WASM volume when WASM is enabled", func() {
			oldWASMflag := featureflags.IsWASMStatsEnabled()
			featureflags.Features.WASMStats = true

			actual := getVolumeSpec("-envoy-config-")
			Expect(actual).To(HaveLen(2))
			Expect(actual[1].Name).To(Equal(envoyStatsWASMVolumeName))

			featureflags.Features.WASMStats = oldWASMflag
		})
	})
})
