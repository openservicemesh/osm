package injector

import (
	"net"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	// outboundPortExclusionListAnnotation is the annotation used for outbound port exclusions
	outboundPortExclusionListAnnotation = "openservicemesh.io/outbound-port-exclusion-list"

	// inboundPortExclusionListAnnotation is the annotation used for inbound port exclusions
	inboundPortExclusionListAnnotation = "openservicemesh.io/inbound-port-exclusion-list"

	// outboundIPRangeExclusionListAnnotation is the annotation used for outbound IP range exclusions
	outboundIPRangeExclusionListAnnotation = "openservicemesh.io/outbound-ip-range-exclusion-list"
)

// getPortExclusionListForPod gets a list of ports to exclude from sidecar traffic interception for the given
// pod and annotation kind.
//
// Ports are excluded from sidecar interception when the pod is explicitly annotated with a single or
// comma separate list of ports.
//
// The kind of exclusion (inbound vs outbound) is determined by the specified annotation.
//
// The function returns an error when it is unable to determine whether ports need to be excluded from outbound sidecar interception.
func getPortExclusionListForPod(pod *corev1.Pod, namespace string, annotation string) ([]int, error) {
	var ports []int

	portsToExcludeStr, ok := pod.Annotations[annotation]
	if !ok {
		// No port exclusion annotation specified
		return ports, nil
	}

	log.Trace().Msgf("Pod with UID %s has port exclusion annotation: '%s:%s'", pod.UID, annotation, portsToExcludeStr)
	portsToExclude := strings.Split(portsToExcludeStr, ",")
	for _, portStr := range portsToExclude {
		portStr := strings.TrimSpace(portStr)
		portVal, err := strconv.Atoi(portStr)
		if err != nil || portVal <= 0 {
			return nil, errors.Errorf("Invalid port value '%s' specified for annotation '%s'", portStr, annotation)
		}
		ports = append(ports, portVal)
	}

	return ports, nil
}

// mergePortExclusionLists merges the pod specific and global port exclusion lists
func mergePortExclusionLists(podSpecificPortExclusionList, globalPortExclusionList []int) []int {
	portExclusionListMap := mapset.NewSet()
	var portExclusionListMerged []int

	// iterate over the global outbound ports to be excluded
	for _, port := range globalPortExclusionList {
		if addedToSet := portExclusionListMap.Add(port); addedToSet {
			portExclusionListMerged = append(portExclusionListMerged, port)
		}
	}

	// iterate over the pod specific ports to be excluded
	for _, port := range podSpecificPortExclusionList {
		if addedToSet := portExclusionListMap.Add(port); addedToSet {
			portExclusionListMerged = append(portExclusionListMerged, port)
		}
	}

	return portExclusionListMerged
}

// getOutboundIPRangeExclusionListForPod gets a list of IP ranges to exclude from sidecar traffic interception for the given
// pod and annotation kind.
//
// IP ranges are excluded from sidecar interception when the pod is explicitly annotated with a single or
// comma separate list of IP CIDR ranges.
//
// The kind of exclusion (inbound vs outbound) is determined by the specified annotation.
//
// The function returns an error when it is unable to determine whether IP ranges need to be excluded from outbound sidecar interception.
func getOutboundIPRangeExclusionListForPod(pod *corev1.Pod, namespace string, annotation string) ([]string, error) {
	ipRangeExclusionsStr, ok := pod.Annotations[annotation]
	if !ok {
		// No port exclusion annotation specified
		return nil, nil
	}

	var ipRanges []string
	log.Trace().Msgf("Pod with UID %s has IP range exclusion annotation: '%s:%s'", pod.UID, annotation, ipRangeExclusionsStr)

	for _, ip := range strings.Split(ipRangeExclusionsStr, ",") {
		ip := strings.TrimSpace(ip)
		if _, _, err := net.ParseCIDR(ip); err != nil {
			return nil, errors.Errorf("Invalid IP range '%s' specified for annotation '%s'", ip, annotation)
		}
		ipRanges = append(ipRanges, ip)
	}

	return ipRanges, nil
}

// mergeIPRangeExclusionLists merges the pod specific and global IP range exclusion lists
func mergeIPRangeExclusionLists(podSpecificExclusionList, globalExclusionList []string) []string {
	ipSet := mapset.NewSet()
	var ipRanges []string

	for _, ip := range podSpecificExclusionList {
		if addedToSet := ipSet.Add(ip); addedToSet {
			ipRanges = append(ipRanges, ip)
		}
	}

	for _, ip := range globalExclusionList {
		if addedToSet := ipSet.Add(ip); addedToSet {
			ipRanges = append(ipRanges, ip)
		}
	}

	return ipRanges
}
