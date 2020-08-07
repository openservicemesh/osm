package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

const versionHelp = `
This command prints out all the version information used by OSM
`

var (
	// BuildDate is date when binary was built
	BuildDate string
	// BuildVersion is the version of binary
	BuildVersion string
	// GitCommit is the commit hash when the binary was built
	GitCommit string
)

// PrintCliVersion prints the version
func PrintCliVersion(out io.Writer) {
	fmt.Fprintf(out, "Version: %s; Commit: %s; Date: %s\n", BuildVersion, GitCommit, BuildDate)
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
