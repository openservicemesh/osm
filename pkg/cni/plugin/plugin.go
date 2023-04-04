// Package plugin implements osm cni plugin.
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	cniv1 "github.com/containernetworking/cni/pkg/types/100"

	"github.com/openservicemesh/osm/pkg/cni/config"
	"github.com/openservicemesh/osm/pkg/cni/util"
)

const (
	// SidecarInjectionAnnotation is the annotation used for sidecar injection
	SidecarInjectionAnnotation = "openservicemesh.io/sidecar-injection"
)

// K8sArgs is the valid CNI_ARGS used for Kubernetes
// The field names need to match exact keys in kubelet args for unmarshalling
type K8sArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString // nolint: revive, stylecheck
	K8S_POD_NAMESPACE          types.UnmarshallableString // nolint: revive, stylecheck
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString // nolint: revive, stylecheck
}

type podInfo struct {
	Containers        []string
	InitContainers    map[string]struct{}
	Labels            map[string]string
	Annotations       map[string]string
	ProxyEnvironments map[string]string
}

func ignore(conf *Config, k8sArgs *K8sArgs) bool {
	ns := string(k8sArgs.K8S_POD_NAMESPACE)
	name := string(k8sArgs.K8S_POD_NAME)
	if ns != "" && name != "" {
		for _, excludeNs := range conf.Kubernetes.ExcludeNamespaces {
			if ns == excludeNs {
				log.Info().Msgf("Pod %s/%s excluded", ns, name)
				return true
			}
		}
		client, err := newKubeClient(*conf)
		if err != nil {
			log.Error().Err(err)
			return true
		}
		pi := &podInfo{}
		for attempt := 1; attempt <= podRetrievalMaxRetries; attempt++ {
			pi, err = getKubePodInfo(client, name, ns)
			if err == nil {
				break
			}
			log.Debug().Msgf("Failed to get %s/%s pod info: %v", ns, name, err)
			time.Sleep(podRetrievalInterval)
		}
		if err != nil {
			log.Error().Msgf("Failed to get %s/%s pod info: %v", ns, name, err)
			return true
		}

		return ignoreMeshlessPod(ns, name, pi)
	}
	log.Debug().Msgf("Not a kubernetes pod")
	return true
}

func ignoreMeshlessPod(namespace, name string, pod *podInfo) bool {
	if len(pod.Containers) > 1 {
		// Check if the pod is annotated for injection
		if podInjectAnnotationExists, injectEnabled, err := isAnnotatedForInjection(pod.Annotations); err != nil {
			log.Warn().Msgf("Pod %s/%s error determining sidecar-injection annotation", namespace, name)
			return true
		} else if podInjectAnnotationExists && !injectEnabled {
			log.Info().Msgf("Pod %s/%s excluded due to sidecar-injection annotation", namespace, name)
			return true
		}

		sidecarExists := false
		for _, container := range pod.Containers {
			if container == `envoy` {
				sidecarExists = true
				break
			}
		}
		if !sidecarExists {
			log.Info().Msgf("Pod %s/%s excluded due to not existing sidecar", namespace, name)
			return true
		}
		return false
	}
	log.Info().Msgf("Pod %s/%s excluded because it only has %d containers", namespace, name, len(pod.Containers))
	return true
}

func isAnnotatedForInjection(annotations map[string]string) (exists bool, enabled bool, err error) {
	inject, ok := annotations[SidecarInjectionAnnotation]
	if !ok {
		return
	}
	exists = true
	switch strings.ToLower(inject) {
	case "enabled", "yes", "true":
		enabled = true
	case "disabled", "no", "false":
		enabled = false
	default:
		err = fmt.Errorf("invalid annotation value for key %q: %s", SidecarInjectionAnnotation, inject)
	}
	return
}

// CmdAdd is the implementation of the cmdAdd interface of CNI plugin
func CmdAdd(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		log.Error().Msgf("osm-cni cmdAdd failed to parse config %v %v", string(args.StdinData), err)
	} else {
		k8sArgs := K8sArgs{}
		if err = types.LoadArgs(args.Args, &k8sArgs); err != nil {
			log.Error().Msgf("osm-cni cmdAdd failed to load args %v %v", string(args.StdinData), err)
		} else {
			if !ignore(conf, &k8sArgs) {
				if util.Exists(config.CNISock) {
					httpc := http.Client{
						Transport: &http.Transport{
							DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
								return net.Dial("unix", config.CNISock)
							},
						},
					}
					bs, _ := json.Marshal(args)
					body := bytes.NewReader(bs)
					if _, err = httpc.Post("http://ecnet-cni"+config.CNICreatePodURL, "application/json", body); err != nil {
						log.Error().Msgf("ecnet-cni cmdAdd failed to post args %v %v", string(args.StdinData), err)
					}
				}
			}
		}
	}

	var result *cniv1.Result
	if conf.PrevResult == nil {
		result = &cniv1.Result{
			CNIVersion: cniv1.ImplementedSpecVersion,
		}
	} else {
		// Pass through the result for the next plugin
		result = conf.PrevResult
	}
	return types.PrintResult(result, conf.CNIVersion)
}

// CmdCheck is the implementation of the cmdCheck interface of CNI plugin
func CmdCheck(*skel.CmdArgs) (err error) {
	return nil
}

// CmdDelete is the implementation of the cmdDelete interface of CNI plugin
func CmdDelete(args *skel.CmdArgs) error {
	if !util.Exists(config.CNISock) {
		return nil
	}

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", config.CNISock)
			},
		},
	}
	bs, _ := json.Marshal(args)
	body := bytes.NewReader(bs)
	_, err := httpc.Post("http://osm-cni"+config.CNIDeletePodURL, "application/json", body)
	log.Error().Msgf("osm-cni cmdDelete failed to parse config %v %v", string(args.StdinData), err)
	return nil
}
