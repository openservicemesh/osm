package main

import (
	"io"

	"github.com/spf13/cobra"
)

const verifyDescription = `
This command consists of multiple subcommands related to verifying
mesh configurations.
`

func newVerifyCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "verify mesh configurations",
		Long:  verifyDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newVerifyConnectivityCmd(stdout, stderr))

	return cmd
}
