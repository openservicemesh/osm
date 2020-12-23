package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/openservicemesh/osm/pkg/version"
)

const versionHelp = `
This command prints out all the version information used by OSM
`

// PrintCliVersion prints the version
func PrintCliVersion(out io.Writer) {
	_, _ = fmt.Fprintf(out, "Version: %s; Commit: %s; Date: %s\n", version.Version, version.GitCommit, version.BuildDate)
}

func newVersionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "osm cli version",
		Long:  versionHelp,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			PrintCliVersion(out)
		},
	}
	return cmd
}
