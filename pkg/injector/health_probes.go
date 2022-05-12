package injector

import (
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	livenessProbePort  = int32(15901)
	readinessProbePort = int32(15902)
	startupProbePort   = int32(15903)
	healthcheckPort    = int32(15904)

	livenessProbePath  = "/osm-liveness-probe"
	readinessProbePath = "/osm-readiness-probe"
	startupProbePath   = "/osm-startup-probe"
	healthcheckPath    = "/osm-healthcheck"
)

var errNoMatchingPort = errors.New("no matching port")

type healthProbe struct {
	path    string
	port    int32
	timeout time.Duration

	// isHTTP corresponds to an httpGet probe with a scheme of HTTP or undefined.
	// This helps inform what kind of Envoy config to add to the pod.
	isHTTP bool

	// isTCPSocket indicates if the probe defines a TCPSocketAction.
	isTCPSocket bool
}

// healthProbes is to serve as an indication whether the given healthProbe has been rewritten
type healthProbes struct {
	liveness, readiness, startup *healthProbe
}

func rewriteHealthProbes(pod *corev1.Pod) healthProbes {
	probes := healthProbes{}
	for idx := range pod.Spec.Containers {
		if probe := rewriteLiveness(&pod.Spec.Containers[idx]); probe != nil {
			probes.liveness = probe
		}
		if probe := rewriteReadiness(&pod.Spec.Containers[idx]); probe != nil {
			probes.readiness = probe
		}
		if probe := rewriteStartup(&pod.Spec.Containers[idx]); probe != nil {
			probes.startup = probe
		}
	}
	return probes
}

func rewriteLiveness(container *corev1.Container) *healthProbe {
	return rewriteProbe(container.LivenessProbe, "liveness", livenessProbePath, livenessProbePort, &container.Ports)
}

func rewriteReadiness(container *corev1.Container) *healthProbe {
	return rewriteProbe(container.ReadinessProbe, "readiness", readinessProbePath, readinessProbePort, &container.Ports)
}

func rewriteStartup(container *corev1.Container) *healthProbe {
	return rewriteProbe(container.StartupProbe, "startup", startupProbePath, startupProbePort, &container.Ports)
}

func rewriteProbe(probe *corev1.Probe, probeType, path string, port int32, containerPorts *[]corev1.ContainerPort) *healthProbe {
	if probe == nil {
		return nil
	}

	originalProbe := &healthProbe{}
	var newPath string
	var definedPort *intstr.IntOrString
	if probe.HTTPGet != nil {
		definedPort = &probe.HTTPGet.Port
		originalProbe.isHTTP = len(probe.HTTPGet.Scheme) == 0 || probe.HTTPGet.Scheme == corev1.URISchemeHTTP
		originalProbe.path = probe.HTTPGet.Path
		if originalProbe.isHTTP {
			probe.HTTPGet.Path = path
			newPath = probe.HTTPGet.Path
		}
	} else if probe.TCPSocket != nil {
		// Transform the TCPSocket probe into a HttpGet probe
		originalProbe.isTCPSocket = true
		probe.HTTPGet = &corev1.HTTPGetAction{
			Port:        probe.TCPSocket.Port,
			Path:        healthcheckPath,
			HTTPHeaders: []corev1.HTTPHeader{},
		}
		newPath = probe.HTTPGet.Path
		definedPort = &probe.HTTPGet.Port
		port = healthcheckPort
		probe.TCPSocket = nil
	} else {
		return nil
	}

	var err error
	originalProbe.port, err = getPort(*definedPort, containerPorts)
	if err != nil {
		log.Error().Err(err).Msgf("Error finding a matching port for %+v on container %+v", *definedPort, containerPorts)
	}
	if originalProbe.isTCPSocket {
		probe.HTTPGet.HTTPHeaders = append(probe.HTTPGet.HTTPHeaders, corev1.HTTPHeader{Name: "Original-Tcp-Port", Value: fmt.Sprint(originalProbe.port)})
	}
	*definedPort = intstr.IntOrString{Type: intstr.Int, IntVal: port}
	originalProbe.timeout = time.Duration(probe.TimeoutSeconds) * time.Second

	log.Debug().Msgf(
		"Rewriting %s probe (:%d%s) to :%d%s",
		probeType,
		originalProbe.port, originalProbe.path,
		port, newPath,
	)

	return originalProbe
}

// getPort returns the int32 of an IntOrString port; It looks for port's name matches in the full list of container ports
func getPort(namedPort intstr.IntOrString, containerPorts *[]corev1.ContainerPort) (int32, error) {
	// Maybe this is not a named port
	intPort := int32(namedPort.IntValue())
	if intPort != 0 {
		return intPort, nil
	}

	if containerPorts == nil {
		return 0, errNoMatchingPort
	}

	// Find an integer match for the name of the port in the list of container ports
	portName := namedPort.String()
	for _, p := range *containerPorts {
		if p.Name != "" && p.Name == portName {
			return p.ContainerPort, nil
		}
	}

	return 0, errNoMatchingPort
}
