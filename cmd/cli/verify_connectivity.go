package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
)

const verifyConnectivityDescription = `
This command consists of multiple subcommands related to verifying
connectivity related configurations.
`

func newVerifyConnectivityCmd(config *action.Configuration, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connectivity",
		Short: "verify connectivity configurations",
		Long:  verifyConnectivityDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newVerifyConnectivityPodToPodCmd(config, stdout, stderr))

	return cmd
}
