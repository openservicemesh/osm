package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Test functions creating Envoy config and rewriting the Pod's health probes to pass through Envoy", func() {
	makePort := func(port int32) intstr.IntOrString {
		return intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: port,
		}
	}

	makeProbe := func(path string, port int32) *v1.Probe {
		return &v1.Probe{
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path: path,
					Port: makePort(port),
				},
			},
			InitialDelaySeconds: 1,
			TimeoutSeconds:      2,
			PeriodSeconds:       3,
			SuccessThreshold:    4,
			FailureThreshold:    5,
		}
	}

	pod := &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				ReadinessProbe: makeProbe("/a", 1),
				LivenessProbe:  makeProbe("/b", 2),
				StartupProbe:   makeProbe("/c", 3),
			}},
		},
	}
	container := &v1.Container{
		Name:           "-some-container-",
		Image:          "-some-container-image-",
		ReadinessProbe: makeProbe("/a/b/c", 1234),
		StartupProbe:   makeProbe("/x/y/z", 3456),
		LivenessProbe:  makeProbe("/k/l/m", 7890),
	}

	containerPorts := &[]v1.ContainerPort{{
		Name:          "-some-port-",
		HostPort:      1234,
		ContainerPort: 34657,
		Protocol:      "http",
		HostIP:        "333.555.666.777",
	}}

	Context("Test rewriteHealthProbes()", func() {
		It("returns the rewritten health probe", func() {
			actual := rewriteHealthProbes(pod)
			expected := healthProbes{
				liveness: &healthProbe{
					path: "/b",
					port: 2,
				},
				readiness: &healthProbe{
					path: "/a",
					port: 1},
				startup: &healthProbe{
					path: "/c",
					port: 3,
				},
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test rewriteLiveness()", func() {
		It("returns the rewritten health probe", func() {
			actual := rewriteLiveness(container)
			expected := &healthProbe{
				path: "/k/l/m",
				port: 7890,
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test rewriteReadiness()", func() {
		It("returns the rewritten health probe", func() {
			actual := rewriteReadiness(container)
			expected := &healthProbe{
				path: "/a/b/c",
				port: 1234,
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test rewriteStartup()", func() {
		It("returns the rewritten health probe", func() {
			actual := rewriteStartup(container)
			expected := &healthProbe{
				path: "/x/y/z",
				port: 3456,
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test rewriteProbe()", func() {
		It("returns the rewritten health probe", func() {
			actual := rewriteProbe(container.StartupProbe, "startup", "/x", 3465, containerPorts)
			expected := &healthProbe{
				path: "/osm-startup-probe",
				port: 15903,
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test getPort()", func() {
		It("returns the port", func() {
			containerPorts := &[]v1.ContainerPort{{
				Name:          "-some-port-",
				ContainerPort: 2344,
			}, {
				Name:          "-some-other-port-",
				ContainerPort: 8877,
			}}

			port1 := intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "-some-port-",
			}

			actual, err := getPort(port1, containerPorts)
			Expect(err).ToNot(HaveOccurred())
			expected := int32(2344)
			Expect(actual).To(Equal(expected))

			port2 := intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 9955,
			}

			actual, err = getPort(port2, containerPorts)
			Expect(err).ToNot(HaveOccurred())
			expected = int32(9955)
			Expect(actual).To(Equal(expected))
		})
	})
})
