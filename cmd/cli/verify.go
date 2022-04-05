package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
)

const verifyDescription = `
This command consists of multiple subcommands related to verifying
mesh configurations.
`

func newVerifyCmd(config *action.Configuration, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "verify mesh configurations",
		Long:  verifyDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newVerifyConnectivityCmd(config, stdout, stderr))

	return cmd
}
