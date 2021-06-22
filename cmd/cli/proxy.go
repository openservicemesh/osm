package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
)

const proxyCmdDescription = `
This command consists of subcommands related to the operations
of the sidecar proxy on pods.
`

func newProxyCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "sidecar proxy operations",
		Long:  proxyCmdDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newProxyGetCmd(config, out))

	return cmd
}
