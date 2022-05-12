package main

import (
	"io"

	"github.com/spf13/cobra"
)

const trafficPolicyDescription = `
This command consists of subcommands related to traffic policies
associated with osm.
`

func newPolicyCmd(stdout io.Writer, _ io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "manage and check traffic policies",
		Long:  trafficPolicyDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newPolicyCheckPods(stdout))
	cmd.AddCommand(newPolicyCheckConflicts(stdout))

	return cmd
}
