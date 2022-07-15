package injector

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
)

const (
	// outboundPortExclusionListAnnotation is the annotation used for outbound port exclusions
	outboundPortExclusionListAnnotation = "openservicemesh.io/outbound-port-exclusion-list"

	// inboundPortExclusionListAnnotation is the annotation used for inbound port exclusions
	inboundPortExclusionListAnnotation = "openservicemesh.io/inbound-port-exclusion-list"

	// outboundIPRangeExclusionListAnnotation is the annotation used for outbound IP range exclusions
	outboundIPRangeExclusionListAnnotation = "openservicemesh.io/outbound-ip-range-exclusion-list"

	// outboundIPRangeInclusionListAnnotation is the annotation used for outbound IP range inclusions
	outboundIPRangeInclusionListAnnotation = "openservicemesh.io/outbound-ip-range-inclusion-list"
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
			return nil, fmt.Errorf("Invalid port value '%s' specified for annotation '%s'", portStr, annotation)
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

// getOutboundIPRangeListForPod returns a list of IP ranges to include/exclude from sidecar traffic interception for the given
// pod and annotation kind.
//
// IP ranges are included/excluded from sidecar interception when the pod is explicitly annotated with a single or
// comma separate list of IP CIDR ranges.
//
// The kind of exclusion (inclusion vs exclusion) is determined by the specified annotation.
//
// The function returns an error when it is unable to determine whether IP ranges need to be excluded from outbound sidecar interception.
func getOutboundIPRangeListForPod(pod *corev1.Pod, namespace string, annotation string) ([]string, error) {
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
			return nil, fmt.Errorf("Invalid IP range '%s' specified for annotation '%s'", ip, annotation)
		}
		ipRanges = append(ipRanges, ip)
	}

	return ipRanges, nil
}

// mergeIPRangeLists merges the pod specific and global IP range (exclusion/inclusion) lists
func mergeIPRangeLists(podSpecific, global []string) []string {
	ipSet := mapset.NewSet()
	var ipRanges []string

	for _, ip := range podSpecific {
		if addedToSet := ipSet.Add(ip); addedToSet {
			ipRanges = append(ipRanges, ip)
		}
	}

	for _, ip := range global {
		if addedToSet := ipSet.Add(ip); addedToSet {
			ipRanges = append(ipRanges, ip)
		}
	}

	return ipRanges
}
