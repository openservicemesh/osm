package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/cli"
	"github.com/openservicemesh/osm/pkg/constants"
)

const proxyCmdDescription = `
This command consists of subcommands related to the operations
of the sidecar proxy on pods.
`

type proxyAdminCmd struct {
	out        io.Writer
	config     *rest.Config
	clientSet  kubernetes.Interface
	query      string
	namespace  string
	pod        string
	localPort  uint16
	outFile    string
	sigintChan chan os.Signal
}

func newProxyCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "sidecar proxy operations",
		Long:  proxyCmdDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newProxyGetCmd(config, out))
	cmd.AddCommand(newProxySetCmd(config, out))

	return cmd
}

func (cmd *proxyAdminCmd) run(reqType string) error {
	response, err := cli.ExecuteEnvoyAdminReq(cmd.clientSet, cmd.config, cmd.namespace, cmd.pod, cmd.localPort, reqType, cmd.query)
	if err != nil {
		return fmt.Errorf("error running proxy cmd: %w", err)
	}

	out := cmd.out // By default, output is written to stdout
	if cmd.outFile != "" {
		fd, err := os.Create(cmd.outFile)
		if err != nil {
			return fmt.Errorf("Error opening file %s: %w", cmd.outFile, err)
		}
		//nolint: errcheck
		//#nosec G307
		defer fd.Close()
		out = fd // write output to file
	}

	_, err = out.Write(response)
	return err
}

// isMeshedPod returns a boolean indicating if the pod is part of a mesh
func isMeshedPod(pod corev1.Pod) bool {
	// osm-controller adds a unique label to each pod that belongs to a mesh
	_, proxyLabelSet := pod.Labels[constants.EnvoyUniqueIDLabelName]
	return proxyLabelSet
}
