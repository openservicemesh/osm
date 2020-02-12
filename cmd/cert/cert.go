package main

import (
	"crypto/rsa"
	"crypto/x509"
	"flag"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/tresor"
)

var (
	flags           = pflag.NewFlagSet(`certificate-tresor`, pflag.ExitOnError)
	host            = flags.String("host", "bookstore.mesh", "host name for the certificate")
	out             = flags.String("out", "", "full path to the certificate PEM file")
	keyout          = flags.String("keyout", "", "full path to the private key PEM file")
	caPEMFileIn     = flags.String("caPEMFileIn", "", "full path to the root cert to be loaded")
	caKeyPEMFileIn  = flags.String("caKeyPEMFileIn", "", "full path to the root cert key to be loaded")
	caPEMFileOut    = flags.String("caPEMFileOut", "", "full path to the root cert to be created")
	caKeyPEMFileOut = flags.String("caKeyPEMFileOut", "", "full path to the root cert key to be created")
	org             = flags.String("org", "ACME Co", "name of the organization for certificate manager")
	validity        = flags.Int("validity", 1, "validity duration of a certificate in MINUTES")
)

func main() {
	defer glog.Flush()
	parseFlags()

	var caPEM tresor.CA
	var caKeyPEM tresor.CAPrivateKey
	var certManager *tresor.CertManager
	var err error

	validityMinutes := time.Duration(*validity) * time.Minute

	if caPEMFileIn != nil && caKeyPEMFileIn != nil && *caPEMFileIn != "" && *caKeyPEMFileIn != "" {
		if certManager, err = tresor.NewCertManagerWithCAFromFile(*caPEMFileIn, *caKeyPEMFileIn, *org, validityMinutes); err != nil {
			glog.Fatal(err)
		}
	} else {
		var ca *x509.Certificate
		var caKey *rsa.PrivateKey
		if caPEM, caKeyPEM, ca, caKey, err = tresor.NewCA(*org, validityMinutes); err != nil {
			glog.Fatal(err)
		}
		certManager, err = tresor.NewCertManagerWithCA(ca, caKey, *org, validityMinutes)
	}

	if err != nil {
		glog.Fatal("Could not instantiate Certificate Manager: ", err)
	}

	cert, err := certManager.IssueCertificate(certificate.CommonName(*host))
	if err != nil {
		glog.Fatal("Error creating a new certificate: ", err)
	}

	if caPEMFileOut != nil && *caPEMFileOut != "" {
		writeFile(*caPEMFileOut, caPEM)
	}
	if caKeyPEMFileOut != nil && *caPEMFileOut != "" {
		writeFile(*caKeyPEMFileOut, caKeyPEM)
	}

	if out != nil && *out != "" {
		writeFile(*out, cert.GetCertificateChain())
	}
	if keyout != nil && *keyout != "" {
		writeFile(*keyout, cert.GetPrivateKey())
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

func parseFlags() {
	if err := flags.Parse(os.Args); err != nil {
		glog.Fatal("Error parsing command line arguments:", err)
	}
	err := flag.CommandLine.Parse([]string{})
	if err != nil {
		glog.Fatal("Could not parse command line parameters: ", err)
	}
}
