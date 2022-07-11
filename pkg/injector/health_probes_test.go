package injector

import (
	"fmt"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openservicemesh/osm/pkg/models"
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
			ProbeHandler: v1.ProbeHandler{
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
			ProbeHandler: v1.ProbeHandler{
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
			ProbeHandler: v1.ProbeHandler{
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
		expected := models.HealthProbes{
			Liveness: &models.HealthProbe{
				Path:    "/b",
				Port:    2,
				IsHTTP:  true,
				Timeout: probeTimeoutDuration,
			},
			Readiness: &models.HealthProbe{
				Path:    "/a",
				Port:    1,
				IsHTTP:  true,
				Timeout: probeTimeoutDuration,
			},
			Startup: &models.HealthProbe{
				Path:    "/c",
				Port:    3,
				IsHTTP:  true,
				Timeout: probeTimeoutDuration,
			},
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteLiveness", func(t *testing.T) {
		actual := rewriteLiveness(container)
		expected := &models.HealthProbe{
			Path:    "/k/l/m",
			Port:    7890,
			IsHTTP:  true,
			Timeout: probeTimeoutDuration,
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteReadiness", func(t *testing.T) {
		actual := rewriteReadiness(container)
		expected := &models.HealthProbe{
			Path:    "/a/b/c",
			Port:    1234,
			IsHTTP:  true,
			Timeout: probeTimeoutDuration,
		}
		tassert.Equal(t, expected, actual)
	})

	t.Run("rewriteStartup", func(t *testing.T) {
		actual := rewriteStartup(container)
		expected := &models.HealthProbe{
			Path:    "/x/y/z",
			Port:    3456,
			IsHTTP:  true,
			Timeout: probeTimeoutDuration,
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
			expected     *models.HealthProbe
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
				expected: &models.HealthProbe{
					Path:    "/x/y/z",
					Port:    0,
					IsHTTP:  true,
					Timeout: probeTimeoutDuration,
				},
			},
			{
				name:    "http",
				probe:   makeHTTPProbe("/x/y/z", 3456),
				newPath: "/x",
				newPort: 3465,
				expected: &models.HealthProbe{
					Path:    "/x/y/z",
					Port:    3456,
					IsHTTP:  true,
					Timeout: probeTimeoutDuration,
				},
			},
			{
				name:    "https",
				probe:   makeHTTPSProbe("/x/y/z", 3456),
				newPath: "/x/y/z",
				newPort: 3465,
				expected: &models.HealthProbe{
					Path:    "/x/y/z",
					Port:    3456,
					IsHTTP:  false,
					Timeout: probeTimeoutDuration,
				},
			},
			{
				name:         "tcp",
				probe:        makeTCPProbe(3456),
				originalPort: 3456,
				newPath:      "/osm-healthcheck",
				newPort:      15904,
				expected: &models.HealthProbe{
					Port:        3456,
					IsHTTP:      false,
					IsTCPSocket: true,
					Timeout:     probeTimeoutDuration,
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
					if test.probe.ProbeHandler.HTTPGet != nil {
						assert.Equal(intstr.FromInt(int(test.newPort)), test.probe.ProbeHandler.HTTPGet.Port)
						assert.Equal(test.newPath, test.probe.ProbeHandler.HTTPGet.Path)
					}
					// After rewrite there should be no TCPSocket probes
					assert.Nil(test.probe.ProbeHandler.TCPSocket)
					if actual != nil && actual.IsTCPSocket {
						expectedHeader := makeOriginalTCPPortHeader(test.originalPort)
						assert.Contains(test.probe.ProbeHandler.HTTPGet.HTTPHeaders, expectedHeader)
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
