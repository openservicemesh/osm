package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Test volume functions", func() {
	Context("Test getVolumeSpec", func() {
		It("creates volume spec", func() {
			actual := getVolumeSpec("-envoy-config-", "-envoy-xds-secret-")
			expected := []v1.Volume{
				{
					Name: "envoy-bootstrap-config-volume",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "-envoy-config-",
							},
						},
					},
				},
				{
					Name: "envoy-xds-secret-volume",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "-envoy-xds-secret-",
						},
					},
				},
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
