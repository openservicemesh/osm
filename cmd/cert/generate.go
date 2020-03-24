package main

import (
	"io"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/tresor"
)

const generateDesc = `
This command generates a new certificate and key.
`

type generateCmd struct {
	output         io.Writer
	validity       int
	caPEMFileIn    string
	caKeyPEMFileIn string
	host           string
	org            string
	out            string
	keyout         string
}

func newGenerateCmd(output io.Writer) *cobra.Command {
	generate := &generateCmd{
		output: output,
	}
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "generate certificate and key",
		Long:  generateDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			return generate.run()
		},
	}

	f := cmd.Flags()
	f.IntVar(&generate.validity, "validity", 525600, "validity duration of a certificate in MINUTES")
	f.StringVar(&generate.caPEMFileIn, "caPEMFileIn", "cert.pem", "full path to the root cert to be loaded")
	f.StringVar(&generate.caKeyPEMFileIn, "caKeyPEMFileIn", "key.pem", "full path to the root cert key to be loaded")
	f.StringVar(&generate.org, "org", "ACME Co", "name of the organization for certificate manager")
	f.StringVar(&generate.host, "host", "bookstore.mesh", "host name for the certificate")
	f.StringVar(&generate.out, "out", "", "full path to the certificate PEM file")
	f.StringVar(&generate.keyout, "keyout", "", "full path to the private key PEM file")

	return cmd
}

func (g *generateCmd) run() error {
	validityMinutes := time.Duration(g.validity) * time.Minute

	var certManager *tresor.CertManager
	var err error

	if g.caPEMFileIn != "" && g.caKeyPEMFileIn != "" {
		if certManager, err = tresor.NewCertManagerWithCAFromFile(g.caPEMFileIn, g.caKeyPEMFileIn, g.org, validityMinutes); err != nil {
			glog.Fatal(err)
		}
	}
	if err != nil {
		glog.Fatal("Could not instantiate Certificate Manager: ", err)
	}

	cert, err := certManager.IssueCertificate(certificate.CommonName(g.host))
	if err != nil {
		glog.Fatal("Error creating a new certificate: ", err)
	}

	writeFile(g.out, cert.GetCertificateChain())
	writeFile(g.keyout, cert.GetPrivateKey())
	return nil
}
