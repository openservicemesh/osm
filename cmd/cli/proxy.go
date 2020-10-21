package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
)

const proxyCmdDescription = `
This command consists of multiple subcommands related to managing the
sidecar proxy on pods.
`

func newProxyCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "manage sidecar proxy",
		Long:  proxyCmdDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newProxyDumpConfig(config, out))

	return cmd
}
