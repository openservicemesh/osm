package main

import (
	goflag "flag"
	"io"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	//"github.com/open-service-mesh/osm/pkg/certificate"
	//"github.com/open-service-mesh/osm/pkg/tresor"
	//"github.com/open-service-mesh/osm/pkg/tresor/pem"
)

var globalUsage = `cert enables you to generate certificates
and keys to use for components of OSM
`

func newRootCmd(args []string, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert",
		Short: "Create and manage certificates and keys with CA",
		Long:  globalUsage,
	}

	cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)
	flags := cmd.PersistentFlags()

	// Add subcommands here
	cmd.AddCommand(
		newBootstrapCmd(out),
		newGenerateCmd(out),
	)

	flags.Parse(args)

	return cmd
}

func main() {
	cmd := newRootCmd(os.Args[1:], os.Stdout)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func writeFile(fileName string, content []byte) {
	if fileName == "" {
		glog.Fatalf("Invalid file name: %+v", fileName)
	}
	keyOut, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		glog.Fatalf("Failed open %s for write: %s", fileName, err)
	}
	bytesWritten, err := keyOut.Write(content)
	if err != nil {
		glog.Fatalf("Failed writing content to file %s: %s", fileName, err)
	}
	glog.Infof("Wrote %d bytes to %s", bytesWritten, fileName)
	if err := keyOut.Close(); err != nil {
		glog.Fatalf("Error closing %s: %s", fileName, err)
	}
}
