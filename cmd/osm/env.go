package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
)

const envHelp = `
This command prints out all the environment information used by OSM
`

func newEnvCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "osm client environment information",
		Long:  envHelp,
		Args:  require.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			envVars := settings.EnvVars()

			// Sort the variables by alphabetical order.
			// This allows for a constant output across calls to 'osm env'.
			var keys []string
			for k := range envVars {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(out, "%s=\"%s\"\n", k, envVars[k])
			}
		},
	}
	return cmd
}
