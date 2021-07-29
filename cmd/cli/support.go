package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
)

const supportCmdDescription = `
This command consists of subcommands related supportability and
associated tooling, such as examining error codes.
`

func newSupportCmd(config *action.Configuration, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "support",
		Short: "supportability tooling",
		Long:  supportCmdDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newSupportErrInfoCmd(stdout))
	cmd.AddCommand(newSupportBugReportCmd(config, stdout, stderr))

	return cmd
}
