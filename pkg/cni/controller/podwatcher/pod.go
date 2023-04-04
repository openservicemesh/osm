package podwatcher

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/cilium/ebpf"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/cni/config"
	"github.com/openservicemesh/osm/pkg/cni/controller/helpers"
	"github.com/openservicemesh/osm/pkg/cni/util"
	"github.com/openservicemesh/osm/pkg/constants"
)

func runLocalPodController(client kubernetes.Interface, stop chan struct{}) error {
	var err error

	if err = helpers.InitLoadPinnedMap(); err != nil {
		return fmt.Errorf("failed to load ebpf maps: %v", err)
	}

	w := newWatcher(createLocalPodController(client))

	if err = w.start(); err != nil {
		return fmt.Errorf("start watcher failed: %v", err)
	}

	log.Info().Msg("Pod watcher Ready")
	if err = helpers.AttachProgs(); err != nil {
		return fmt.Errorf("failed to attach ebpf programs: %v", err)
	}
	if config.EnableCNI {
		<-stop
	} else {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
		<-ch
	}
	w.shutdown()

	if err = helpers.UnLoadProgs(); err != nil {
		return fmt.Errorf("unload failed: %v", err)
	}
	log.Info().Msg("Pod watcher Down")
	return nil
}

func createLocalPodController(client kubernetes.Interface) watcher {
	localName, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return watcher{
		Client:          client,
		CurrentNodeName: localName,
		OnAddFunc:       addFunc,
		OnUpdateFunc:    updateFunc,
		OnDeleteFunc:    deleteFunc,
	}
}

const maxItemLen = 20 // todo changeme

type cidr struct {
	net  uint32 // network order
	mask uint8
	_    [3]uint8 // pad
}

type podConfig struct {
	statusPort       uint16
	_                uint16 // pad
	excludeOutRanges [maxItemLen]cidr
	includeOutRanges [maxItemLen]cidr
	includeInPorts   [maxItemLen]uint16
	includeOutPorts  [maxItemLen]uint16
	excludeInPorts   [maxItemLen]uint16
	excludeOutPorts  [maxItemLen]uint16
}

func isInjectedSidecar(pod *v1.Pod) bool {
	if _, found := pod.Labels[constants.EnvoyUniqueIDLabelName]; found {
		return true
	}
	return false
}

func addFunc(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok || len(pod.Status.PodIP) == 0 {
		return
	}
	if !isInjectedSidecar(pod) {
		return
	}
	log.Debug().Msgf("got pod updated %s/%s", pod.Namespace, pod.Name)

	_ip, _ := util.IP2Pointer(pod.Status.PodIP)
	log.Info().Msgf("update osm_pod_fib with ip: %s", pod.Status.PodIP)
	p := podConfig{}
	parsePodConfigFromAnnotations(pod.Annotations, &p)
	err := helpers.GetPodFibMap().Update(_ip, &p, ebpf.UpdateAny)
	if err != nil {
		log.Error().Msgf("update osm_pod_fib %s error: %v", pod.Status.PodIP, err)
	}
}

func getPortsFromString(v string) []uint16 {
	var ports []uint16
	for _, vv := range strings.Split(v, ",") {
		if p := strings.TrimSpace(vv); p != "" {
			port, err := strconv.ParseUint(vv, 10, 16)
			if err == nil {
				ports = append(ports, uint16(port))
			}
		}
	}
	return ports
}

func getIPRangesFromString(v string) []cidr {
	var ranges []cidr
	for _, vv := range strings.Split(v, ",") {
		if vv == "*" {
			ranges = append(ranges, cidr{
				net:  0,
				mask: 0,
			})
			continue
		}
		if p := strings.TrimSpace(vv); p != "" {
			_, n, err := net.ParseCIDR(vv)
			if err != nil {
				log.Error().Msgf("parse cidr from %s error: %v", vv, err)
				continue
			}
			c := cidr{}
			ones, _ := n.Mask.Size()
			c.mask = uint8(ones)
			if len(n.IP) == 16 {
				//#nosec G103
				c.net = *(*uint32)(unsafe.Pointer(&n.IP[12]))
			} else {
				//#nosec G103
				c.net = *(*uint32)(unsafe.Pointer(&n.IP[0]))
			}
			ranges = append(ranges, c)
		}
	}
	return ranges
}

func parsePodConfigFromAnnotations(annotations map[string]string, pod *podConfig) {
	statusPort := 15021
	if v, ok := annotations["openservicemesh.io/port"]; ok {
		vv, err := strconv.ParseUint(v, 10, 16)
		if err == nil {
			statusPort = int(vv)
		}
	}
	pod.statusPort = uint16(statusPort)
	excludeInboundPorts := []uint16{15000, 15001, 15003, 15010, 15021, 15050, 15128, 15901, 15902, 15903, 15904}
	if v, ok := annotations["openservicemesh.io/inbound-port-exclusion-list"]; ok {
		excludeInboundPorts = append(excludeInboundPorts, getPortsFromString(v)...)
	}
	if len(excludeInboundPorts) > 0 {
		for i, p := range excludeInboundPorts {
			if i >= maxItemLen {
				break
			}
			pod.excludeInPorts[i] = p
		}
	}
	if v, ok := annotations["openservicemesh.io/outbound-port-exclusion-list"]; ok {
		excludeOutboundPorts := getPortsFromString(v)
		if len(excludeOutboundPorts) > 0 {
			for i, p := range excludeOutboundPorts {
				if i >= maxItemLen {
					break
				}
				pod.excludeOutPorts[i] = p
			}
		}
	}

	if v, ok := annotations["openservicemesh.io/inbound-port-inclusion-list"]; ok {
		includeInboundPorts := getPortsFromString(v)
		if len(includeInboundPorts) > 0 {
			for i, p := range includeInboundPorts {
				if i >= maxItemLen {
					break
				}
				pod.includeInPorts[i] = p
			}
		}
	}
	if v, ok := annotations["openservicemesh.io/outbound-port-inclusion-list"]; ok {
		includeOutboundPorts := getPortsFromString(v)
		if len(includeOutboundPorts) > 0 {
			for i, p := range includeOutboundPorts {
				if i >= maxItemLen {
					break
				}
				pod.includeOutPorts[i] = p
			}
		}
	}

	if v, ok := annotations["openservicemesh.io/outbound-ip-range-exclusion-list"]; ok {
		excludeOutboundIPRanges := getIPRangesFromString(v)
		if len(excludeOutboundIPRanges) > 0 {
			for i, p := range excludeOutboundIPRanges {
				if i >= maxItemLen {
					break
				}
				pod.excludeOutRanges[i] = p
			}
		}
	}
	if v, ok := annotations["openservicemesh.io/outbound-ip-range-inclusion-list"]; ok {
		includeOutboundIPRanges := getIPRangesFromString(v)
		if len(includeOutboundIPRanges) > 0 {
			for i, p := range includeOutboundIPRanges {
				if i >= maxItemLen {
					break
				}
				pod.includeOutRanges[i] = p
			}
		}
	}
}

func updateFunc(old, cur interface{}) {
	oldPod, ok := old.(*v1.Pod)
	if !ok {
		return
	}
	curPod, ok := cur.(*v1.Pod)
	if !ok {
		return
	}
	if oldPod.Status.PodIP != curPod.Status.PodIP {
		// only care about ip changes
		addFunc(cur)
	}
}

func deleteFunc(obj interface{}) {
	if pod, ok := obj.(*v1.Pod); ok {
		log.Debug().Msgf("got pod delete %s/%s", pod.Namespace, pod.Name)
		_ip, _ := util.IP2Pointer(pod.Status.PodIP)
		_ = helpers.GetPodFibMap().Delete(_ip)
	}
}
