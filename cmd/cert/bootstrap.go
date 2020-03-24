package main

import (
	"io"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/open-service-mesh/osm/pkg/tresor"
)

const bootstrapDesc = `
This command bootstraps a CA and generates a root certificate
and root key.
`

type bootstrapCmd struct {
	out             io.Writer
	validity        int
	caPEMFileOut    string
	caKeyPEMFileOut string
	org             string
}

func newBootstrapCmd(out io.Writer) *cobra.Command {
	bootstrap := &bootstrapCmd{
		out: out,
	}
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "bootstrap CA",
		Long:  bootstrapDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			return bootstrap.run()
		},
	}

	f := cmd.Flags()
	f.IntVar(&bootstrap.validity, "validity", 525600, "validity duration of a certificate in MINUTES")
	f.StringVar(&bootstrap.caPEMFileOut, "caPEMFileOut", "root-cert.pem", "full path to the root cert to be created")
	f.StringVar(&bootstrap.caKeyPEMFileOut, "caKeyPEMFileOut", "root-key.pem", "full path to the root cert key to be created")
	f.StringVar(&bootstrap.org, "org", "ACME Co", "name of the organization for certificate manager")

	return cmd
}

func (b *bootstrapCmd) run() error {
	validityMinutes := time.Duration(b.validity) * time.Minute

	caPEM, caKeyPEM, _, _, err := tresor.NewCA(b.org, validityMinutes)
	if err != nil {
		glog.Fatal(err)
	}

	writeFile(b.caPEMFileOut, caPEM)
	writeFile(b.caKeyPEMFileOut, caKeyPEM)

	return nil
}
