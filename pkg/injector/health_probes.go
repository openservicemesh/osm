package injector

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// TODO(draychev): Dynamically generate init/init-iptables.sh from these constants: https://github.com/openservicemesh/osm/issues/2243
	livenessProbePort  = int32(15901)
	readinessProbePort = int32(15902)
	startupProbePort   = int32(15903)

	livenessProbePath  = "/osm-liveness-probe"
	readinessProbePath = "/osm-readiness-probe"
	startupProbePath   = "/osm-startup-probe"
)

var errNoMatchingPort = errors.New("no matching port")

type healthProbe struct {
	path string
	port int32
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
	if probe.HTTPGet == nil {
		return nil
	}

	originalPort, err := getPort(probe.HTTPGet.Port, containerPorts)
	if err != nil {
		log.Err(err).Msgf("Error finding a matching port for %+v on container %+v", probe.HTTPGet.Port, containerPorts)
	}
	originalPath := probe.HTTPGet.Path

	probe.HTTPGet.Port = intstr.IntOrString{Type: intstr.Int, IntVal: port}
	probe.HTTPGet.Path = path

	log.Debug().Msgf(
		"Rewriting %s probe (:%d%s) to :%d%s",
		probeType,
		originalPort, originalPath,
		probe.HTTPGet.Port.IntValue(), probe.HTTPGet.Path,
	)

	return &healthProbe{
		port: originalPort,
		path: originalPath,
	}
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
