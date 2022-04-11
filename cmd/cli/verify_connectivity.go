package main

import (
	"io"

	"github.com/spf13/cobra"
)

const verifyConnectivityDescription = `
This command consists of multiple subcommands related to verifying
connectivity related configurations.
`

func newVerifyConnectivityCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connectivity",
		Short: "verify connectivity configurations",
		Long:  verifyConnectivityDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newVerifyConnectivityPodToPodCmd(stdout, stderr))

	return cmd
}
