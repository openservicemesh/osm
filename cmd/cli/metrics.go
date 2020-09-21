package main

import (
	"io"

	"github.com/spf13/cobra"
)

const metricsDescription = `
This command consists of multiple subcommands related to managing metrics
associated with osm.
`

func newMetricsCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "manage metrics",
		Long:  metricsDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newMetricsEnable(out))
	cmd.AddCommand(newMetricsDisable(out))

	return cmd
}
