package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	smiAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	osmConfigClient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/cli"
)

const trafficPolicyCheckDescription = `
This command will check whether a given source pod is allowed to communicate
(send traffic) to a given destination pod by an SMI TrafficTarget policy or
in lieu of the mesh operating in permissive traffic policy mode.
`

const trafficPolicyCheckExample = `
# To check if pod 'bookbuyer-client' in the 'bookbuyer' namespace can send traffic to pod 'bookstore-server' in the 'bookstore' namespace
osm policy check-pods bookbuyer/bookbuyer-client bookstore/bookstore-server

# If the pod belongs to the default namespace, the namespace can be omitted with the flags
# To check if pod 'bookbuyer-client' in the 'default' namespace can send traffic to pod 'bookstore-server' in the 'default' namespace
osm policy check-pods bookbuyer-client bookstore-server
`

const (
	namespaceSeparator       = "/"
	defaultOsmMeshConfigName = "osm-mesh-config"
)

type trafficPolicyCheckCmd struct {
	out              io.Writer
	sourcePod        string
	destinationPod   string
	clientSet        kubernetes.Interface
	smiAccessClient  smiAccessClient.Interface
	meshConfigClient osmConfigClient.Interface
	restConfig       *rest.Config
}

func newPolicyCheckPods(out io.Writer) *cobra.Command {
	trafficPolicyCheckCmd := &trafficPolicyCheckCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "check-pods SOURCE_POD DESTINATION_POD",
		Short: "check if two pods are authorized to communicate via a traffic policy",
		Long:  trafficPolicyCheckDescription,
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			trafficPolicyCheckCmd.sourcePod = args[0]
			trafficPolicyCheckCmd.destinationPod = args[1]

			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}

			trafficPolicyCheckCmd.restConfig = config

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			trafficPolicyCheckCmd.clientSet = clientset

			accessClient, err := smiAccessClient.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not initialize SMI Access client: %w", err)
			}
			trafficPolicyCheckCmd.smiAccessClient = accessClient

			configClient, err := osmConfigClient.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not initialize OSM Config client: %w", err)
			}
			trafficPolicyCheckCmd.meshConfigClient = configClient

			return trafficPolicyCheckCmd.run()
		},
		Example: trafficPolicyCheckExample,
	}

	return cmd
}

func (cmd *trafficPolicyCheckCmd) run() error {
	// Validate input for options
	srcNs, srcPodName, err := unmarshalNamespacedPod(cmd.sourcePod)
	if err != nil {
		return fmt.Errorf("Invalid argument specified for the source pod [%s/%s]: %w", srcNs, srcPodName, err)
	}

	dstNs, dstPodName, err := unmarshalNamespacedPod(cmd.destinationPod)
	if err != nil {
		return fmt.Errorf("Invalid argument specified for the destination pod [%s/%s]: %w", dstNs, dstPodName, err)
	}

	srcPod, err := cmd.getMeshedPod(srcNs, srcPodName)
	if err != nil {
		return err
	}
	dstPod, err := cmd.getMeshedPod(dstNs, dstPodName)
	if err != nil {
		return err
	}

	return cmd.checkTrafficPolicy(srcPod, dstPod)
}

func (cmd *trafficPolicyCheckCmd) checkTrafficPolicy(srcPod, dstPod *corev1.Pod) error {
	osmNamespace := settings.Namespace()

	// Check if permissive mode is enabled, in which case every meshed pod is allowed to communicate with each other
	if permissiveMode, err := cmd.isPermissiveModeEnabled(); err != nil {
		return fmt.Errorf("Error checking if permissive mode is enabled: %w", err)
	} else if permissiveMode {
		fmt.Fprintf(cmd.out, "[+] Permissive mode enabled for mesh operated by osm-controller running in '%s' namespace\n\n "+
			"[+] Pod '%s/%s' is allowed to communicate to pod '%s/%s'\n",
			osmNamespace, srcPod.Namespace, srcPod.Name, dstPod.Namespace, dstPod.Name)
		return nil
	}

	// SMI traffic policy mode
	fmt.Fprintf(cmd.out, "[+] SMI traffic policy mode enabled for mesh operated by osm-controller running in %s namespace\n\n", osmNamespace)
	trafficTargets, err := cmd.smiAccessClient.AccessV1alpha3().TrafficTargets(dstPod.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Error listing SMI TrafficTarget policies: %w", err)
	}

	var foundTrafficTarget bool
	for _, trafficTarget := range trafficTargets.Items {
		spec := trafficTarget.Spec

		// Map traffic targets to the given pods
		if !cli.DoesTargetRefDstPod(spec, dstPod) {
			continue
		}
		// The TrafficTarget destination is associated to 'dstPod', check if 'srcPod` is an allowed source to this destination
		if cli.DoesTargetRefSrcPod(spec, srcPod) {
			foundTrafficTarget = true
			fmt.Fprintf(cmd.out, "[+] Pod '%s/%s' is allowed to communicate to pod '%s/%s' via the SMI TrafficTarget policy %q in namespace %s\n",
				srcPod.Namespace, srcPod.Name, dstPod.Namespace, dstPod.Name, trafficTarget.Name, trafficTarget.Namespace)
		}
	}

	if !foundTrafficTarget {
		fmt.Fprintf(cmd.out, "[+] Pod '%s/%s' is not allowed to communicate to pod '%s/%s', missing SMI TrafficTarget policy\n",
			srcPod.Namespace, srcPod.Name, dstPod.Namespace, dstPod.Name)
	}

	return nil
}

func (cmd *trafficPolicyCheckCmd) getMeshedPod(namespace, podName string) (*corev1.Pod, error) {
	// Validate the pods
	pod, err := cmd.clientSet.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Could not find pod %s in namespace %s", podName, namespace)
	}
	if !isMeshedPod(*pod) {
		return nil, fmt.Errorf("Pod %s in namespace %s is not a part of a mesh", podName, namespace)
	}
	return pod, nil
}

func (cmd *trafficPolicyCheckCmd) isPermissiveModeEnabled() (bool, error) {
	osmNamespace := settings.Namespace()

	meshConfig, err := cmd.meshConfigClient.ConfigV1alpha2().MeshConfigs(osmNamespace).Get(context.TODO(), defaultOsmMeshConfigName, metav1.GetOptions{})

	if err != nil {
		return false, fmt.Errorf("Error fetching MeshConfig %s: %w", defaultOsmMeshConfigName, err)
	}
	return meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode, nil
}

func unmarshalNamespacedPod(namespacedPod string) (namespace string, podName string, err error) {
	if namespacedPod == "" {
		err = fmt.Errorf("Pod name should be of the form <namespace/pod>, or <pod> for default namespace, cannot be empty")
		return
	}
	chunks := strings.Split(namespacedPod, namespaceSeparator)
	if len(chunks) == 1 {
		namespace = metav1.NamespaceDefault
		podName = chunks[0]
	} else if len(chunks) == 2 {
		namespace = chunks[0]
		podName = chunks[1]
	} else {
		err = fmt.Errorf("Pod name should be of the form <namespace/pod>, or <pod> for default namespace, got: %s", namespacedPod)
	}
	return
}
