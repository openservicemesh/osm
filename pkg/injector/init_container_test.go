package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Test volume functions", func() {
	Context("Test getInitContainerSpec", func() {
		It("creates volume spec", func() {
			container := &InitContainer{
				Name:  "-container-name-",
				Image: "-init-container-image-",
			}
			actual, err := getInitContainerSpec(container)
			Expect(err).ToNot(HaveOccurred())
			expected := v1.Container{
				Name:       "-container-name-",
				Image:      "-init-container-image-",
				WorkingDir: "",
				Env: []v1.EnvVar{
					{
						Name:  "OSM_PROXY_UID",
						Value: "1337",
					},
					{
						Name:  "OSM_ENVOY_INBOUND_PORT",
						Value: "15003",
					},
					{
						Name:  "OSM_ENVOY_OUTBOUND_PORT",
						Value: "15001",
					},
				},
				Resources: v1.ResourceRequirements{},
				SecurityContext: &v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{
							"NET_ADMIN",
						},
					},
					Privileged: nil,
				},
				Stdin:     false,
				StdinOnce: false,
				TTY:       false,
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
