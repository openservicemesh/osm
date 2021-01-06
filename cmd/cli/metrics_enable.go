package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

const metricsEnableDescription = `
This command will enable metrics scraping on all pods belonging to the given
namespace or set of namespaces. Newly created pods belonging to namespaces that
are enabled for metrics will be automatically enabled with metrics.

The command does not deploy a metrics collection service such as Prometheus.
`

type metricsEnableCmd struct {
	out        io.Writer
	namespaces []string
	clientSet  kubernetes.Interface
}

func newMetricsEnable(out io.Writer) *cobra.Command {
	enableCmd := &metricsEnableCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "enable ...",
		Short: "enable metrics",
		Long:  metricsEnableDescription,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			enableCmd.clientSet = clientset
			return enableCmd.run()
		},
	}

	//add mesh name flag
	f := cmd.Flags()
	f.StringSliceVar(&enableCmd.namespaces, "namespace", []string{}, "One or more namespaces to enable metrics on")

	return cmd
}

func (cmd *metricsEnableCmd) run() error {
	// Add metrics annotation on namespaces
	for _, ns := range cmd.namespaces {
		ns = strings.TrimSpace(ns)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		namespace, err := cmd.clientSet.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err != nil {
			return errors.Errorf("Failed to retrieve namespace [%s]: %v", ns, err)
		}

		// Check if the namespace belongs to a mesh, if not return an error
		monitored, err := isMonitoredNamespace(*namespace, getMeshNames(cmd.clientSet))
		if err != nil {
			return err
		}
		if !monitored {
			return errors.Errorf("Namespace [%s] does not belong to a mesh, missing annotation %q",
				ns, constants.OSMKubeResourceMonitorAnnotation)
		}

		// Patch the namespace with metrics annotation.
		// osm-controller uses this annotation to automatically enable new pods for metrics scraping.
		patch := fmt.Sprintf(`
{
	"metadata": {
		"annotations": {
			"%s": "enabled"
		}
	}
}`, constants.MetricsAnnotation)

		_, err = cmd.clientSet.CoreV1().Namespaces().Patch(ctx, ns, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")
		if err != nil {
			return errors.Errorf("Failed to enable metrics in namespace [%s]: %v", ns, err)
		}

		// For existing pods in this namespace that are already part of the mesh, add the prometheus
		// scraping annotations.
		if err := cmd.enableMetricsForPods(ns); err != nil {
			return errors.Errorf("Failed to enable metrics for existing pod in namespace [%s]: %v", ns, err)
		}

		fmt.Fprintf(cmd.out, "Metrics successfully enabled in namespace [%s]\n", ns)
	}

	return nil
}

// enableMetricsForPods enables metrics for existing pods in the given namespace
func (cmd *metricsEnableCmd) enableMetricsForPods(namespace string) error {
	listOptions := metav1.ListOptions{
		// Matches on pods which are already a part of the mesh, which contain the Envoy ID label
		LabelSelector: constants.EnvoyUniqueIDLabelName,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	podList, err := cmd.clientSet.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return err
	}

	for _, pod := range podList.Items {
		// Patch existing pods in this namespace with annotations used for metrics scraping
		patch := fmt.Sprintf(`
{
	"metadata": {
		"annotations": {
			"%s": "true",
			"%s": "%d",
			"%s": "%s"
		}
	}
}`, constants.PrometheusScrapeAnnotation, constants.PrometheusPortAnnotation, constants.EnvoyPrometheusInboundListenerPort,
			constants.PrometheusPathAnnotation, constants.PrometheusScrapePath)

		_, err = cmd.clientSet.CoreV1().Pods(namespace).Patch(ctx, pod.Name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")
		if err != nil {
			return err
		}
	}

	return nil
}
