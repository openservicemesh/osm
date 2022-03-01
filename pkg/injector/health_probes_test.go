package injector

import (
	"fmt"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestRewriteProbe(t *testing.T) {
	const probeTimeoutSeconds = 2
	const probeTimeoutDuration = probeTimeoutSeconds * time.Second
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
			TimeoutSeconds:      probeTimeoutSeconds,
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
			TimeoutSeconds:      probeTimeoutSeconds,
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
			TimeoutSeconds:      probeTimeoutSeconds,
			PeriodSeconds:       3,
			SuccessThreshold:    4,
			FailureThreshold:    5,
		}
	}

	makeOriginalTCPPortHeader := func(port int32) v1.HTTPHeader {
		return v1.HTTPHeader{
			Name:  "Original-Tcp-Port",
			Value: fmt.Sprint(port),
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
				path:    "/b",
				port:    2,
				isHTTP:  true,
				timeout: probeTimeoutDuration,
			},
			readiness: &healthProbe{
				path:    "/a",
				port:    1,
				isHTTP:  true,
				timeout: probeTimeoutDuration,
			},
			startup: &healthProbe{
				path:    "/c",
				port:    3,
				isHTTP:  true,
				timeout: probeTimeoutDuration,
			},
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteLiveness", func(t *testing.T) {
		actual := rewriteLiveness(container)
		expected := &healthProbe{
			path:    "/k/l/m",
			port:    7890,
			isHTTP:  true,
			timeout: probeTimeoutDuration,
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteReadiness", func(t *testing.T) {
		actual := rewriteReadiness(container)
		expected := &healthProbe{
			path:    "/a/b/c",
			port:    1234,
			isHTTP:  true,
			timeout: probeTimeoutDuration,
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteStartup", func(t *testing.T) {
		actual := rewriteStartup(container)
		expected := &healthProbe{
			path:    "/x/y/z",
			port:    3456,
			isHTTP:  true,
			timeout: probeTimeoutDuration,
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteProbe", func(t *testing.T) {
		tests := []struct {
			name         string
			probe        *v1.Probe
			newPath      string
			originalPort int32
			newPort      int32
			expected     *healthProbe
		}{
			{
				name:     "nil",
				probe:    nil,
				expected: nil,
			},
			{
				name:     "no http or tcp",
				probe:    &v1.Probe{},
				expected: nil,
			},
			{
				name:  "getPort() error",
				probe: makeHTTPProbe("/x/y/z", 0),
				expected: &healthProbe{
					path:    "/x/y/z",
					port:    0,
					isHTTP:  true,
					timeout: probeTimeoutDuration,
				},
			},
			{
				name:    "http",
				probe:   makeHTTPProbe("/x/y/z", 3456),
				newPath: "/x",
				newPort: 3465,
				expected: &healthProbe{
					path:    "/x/y/z",
					port:    3456,
					isHTTP:  true,
					timeout: probeTimeoutDuration,
				},
			},
			{
				name:    "https",
				probe:   makeHTTPSProbe("/x/y/z", 3456),
				newPath: "/x/y/z",
				newPort: 3465,
				expected: &healthProbe{
					path:    "/x/y/z",
					port:    3456,
					isHTTP:  false,
					timeout: probeTimeoutDuration,
				},
			},
			{
				name:         "tcp",
				probe:        makeTCPProbe(3456),
				originalPort: 3456,
				newPath:      "/osm-healthcheck",
				newPort:      15904,
				expected: &healthProbe{
					port:        3456,
					isHTTP:      false,
					isTCPSocket: true,
					timeout:     probeTimeoutDuration,
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
				if test.probe != nil {
					if test.probe.Handler.HTTPGet != nil {
						assert.Equal(intstr.FromInt(int(test.newPort)), test.probe.Handler.HTTPGet.Port)
						assert.Equal(test.newPath, test.probe.Handler.HTTPGet.Path)
					}
					// After rewrite there should be no TCPSocket probes
					assert.Nil(test.probe.Handler.TCPSocket)
					if actual != nil && actual.isTCPSocket {
						expectedHeader := makeOriginalTCPPortHeader(test.originalPort)
						assert.Contains(test.probe.Handler.HTTPGet.HTTPHeaders, expectedHeader)
					}
				}
			})
		}
	})
}

func TestGetPort(t *testing.T) {
	tests := []struct {
		name           string
		port           intstr.IntOrString
		containerPorts *[]v1.ContainerPort
		expectedPort   int32
		expectedErr    error
	}{
		{
			name:           "no container ports",
			port:           intstr.FromString("-some-port-"),
			containerPorts: &[]v1.ContainerPort{},
			expectedErr:    errNoMatchingPort,
		},
		{
			name: "named port",
			port: intstr.FromString("-some-port-"),
			containerPorts: &[]v1.ContainerPort{
				{Name: "-some-port-", ContainerPort: 2344},
				{Name: "-some-other-port-", ContainerPort: 8877},
			},
			expectedPort: 2344,
		},
		{
			name: "numbered port",
			port: intstr.FromInt(9955),
			containerPorts: &[]v1.ContainerPort{
				{Name: "-some-port-", ContainerPort: 2344},
				{Name: "-some-other-port-", ContainerPort: 8877},
			},
			expectedPort: 9955,
		},
		{
			name: "no matching named ports",
			port: intstr.FromString("-another-port-"),
			containerPorts: &[]v1.ContainerPort{
				{Name: "-some-port-", ContainerPort: 2344},
				{Name: "-some-other-port-", ContainerPort: 8877},
			},
			expectedErr: errNoMatchingPort,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			actual, err := getPort(test.port, test.containerPorts)

			if test.expectedErr != nil {
				assert.ErrorIs(err, errNoMatchingPort)
				return
			}

			assert.Nil(err)
			assert.Equal(test.expectedPort, actual)
		})
	}
}
