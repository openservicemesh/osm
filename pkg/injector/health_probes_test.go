package injector

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestRewriteProbe(t *testing.T) {
	makePort := func(port int32) intstr.IntOrString {
		return intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: port,
		}
	}

	makeHTTPProbe := func(path string, port int32) *v1.Probe {
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

	makeHTTPSProbe := func(path string, port int32) *v1.Probe {
		return &v1.Probe{
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path:   path,
					Port:   makePort(port),
					Scheme: v1.URISchemeHTTPS,
				},
			},
			InitialDelaySeconds: 1,
			TimeoutSeconds:      2,
			PeriodSeconds:       3,
			SuccessThreshold:    4,
			FailureThreshold:    5,
		}
	}

	makeTCPProbe := func(port int32) *v1.Probe {
		return &v1.Probe{
			Handler: v1.Handler{
				TCPSocket: &v1.TCPSocketAction{
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
				ReadinessProbe: makeHTTPProbe("/a", 1),
				LivenessProbe:  makeHTTPProbe("/b", 2),
				StartupProbe:   makeHTTPProbe("/c", 3),
			}},
		},
	}
	container := &v1.Container{
		Name:           "-some-container-",
		Image:          "-some-container-image-",
		ReadinessProbe: makeHTTPProbe("/a/b/c", 1234),
		StartupProbe:   makeHTTPProbe("/x/y/z", 3456),
		LivenessProbe:  makeHTTPProbe("/k/l/m", 7890),
	}

	t.Run("rewriteHealthProbes", func(t *testing.T) {
		actual := rewriteHealthProbes(pod)
		expected := healthProbes{
			liveness: &healthProbe{
				path:   "/b",
				port:   2,
				isHTTP: true,
			},
			readiness: &healthProbe{
				path:   "/a",
				port:   1,
				isHTTP: true,
			},
			startup: &healthProbe{
				path:   "/c",
				port:   3,
				isHTTP: true,
			},
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteLiveness", func(t *testing.T) {
		actual := rewriteLiveness(container)
		expected := &healthProbe{
			path:   "/k/l/m",
			port:   7890,
			isHTTP: true,
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteReadiness", func(t *testing.T) {
		actual := rewriteReadiness(container)
		expected := &healthProbe{
			path:   "/a/b/c",
			port:   1234,
			isHTTP: true,
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteStartup", func(t *testing.T) {
		actual := rewriteStartup(container)
		expected := &healthProbe{
			path:   "/x/y/z",
			port:   3456,
			isHTTP: true,
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteProbe", func(t *testing.T) {
		tests := []struct {
			name     string
			probe    *v1.Probe
			newPath  string
			newPort  int32
			expected *healthProbe
		}{
			{
				name:    "http",
				probe:   makeHTTPProbe("/x/y/z", 3456),
				newPath: "/x",
				newPort: 3465,
				expected: &healthProbe{
					path:   "/x/y/z",
					port:   3456,
					isHTTP: true,
				},
			},
			{
				name:    "https",
				probe:   makeHTTPSProbe("/x/y/z", 3456),
				newPath: "/x/y/z",
				newPort: 3465,
				expected: &healthProbe{
					path:   "/x/y/z",
					port:   3456,
					isHTTP: false,
				},
			},
			{
				name:    "tcp",
				probe:   makeTCPProbe(3456),
				newPort: 3465,
				expected: &healthProbe{
					port:   3456,
					isHTTP: false,
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert := tassert.New(t)

				// probeType left blank here because its value is only logged.
				// containerPorts are not defined here because it's only used
				// in getPort(), which is tested below.
				actual := rewriteProbe(test.probe, "", test.newPath, test.newPort, nil)
				assert.Equal(test.expected, actual)

				// Verify the probe was modified correctly
				if test.probe.Handler.HTTPGet != nil {
					assert.Equal(test.probe.Handler.HTTPGet.Port, intstr.FromInt(int(test.newPort)))
					assert.Equal(test.probe.Handler.HTTPGet.Path, test.newPath)
				}
				if test.probe.Handler.TCPSocket != nil {
					assert.Equal(test.probe.Handler.TCPSocket.Port, intstr.FromInt(int(test.newPort)))
				}
			})
		}
	})
}

func TestGetPort(t *testing.T) {
	containerPorts := &[]v1.ContainerPort{
		{
			Name:          "-some-port-",
			ContainerPort: 2344,
		},
		{
			Name:          "-some-other-port-",
			ContainerPort: 8877,
		},
	}

	tests := []struct {
		name     string
		port     intstr.IntOrString
		expected int32
	}{
		{
			name:     "named port",
			port:     intstr.FromString("-some-port-"),
			expected: 2344,
		},
		{
			name:     "numbered port",
			port:     intstr.FromInt(9955),
			expected: 9955,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			actual, err := getPort(test.port, containerPorts)
			assert.Nil(err)
			assert.Equal(test.expected, actual)
		})
	}
}
