package main

import (
	"crypto/rsa"
	"crypto/x509"
	"flag"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/tresor"
	"github.com/open-service-mesh/osm/pkg/tresor/pem"
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
	validity        = flags.Int("validity", 525600, "validity duration of a certificate in MINUTES")
)

func main() {
	parseFlags()

	var caPEM pem.RootCertificate
	var caKeyPEM pem.RootPrivateKey
	var certManager *tresor.CertManager
	var err error

	validityMinutes := time.Duration(*validity) * time.Minute

	if caPEMFileIn != nil && caKeyPEMFileIn != nil && *caPEMFileIn != "" && *caKeyPEMFileIn != "" {
		if certManager, err = tresor.NewCertManagerWithCAFromFile(*caPEMFileIn, *caKeyPEMFileIn, *org, validityMinutes); err != nil {
			log.Fatal().Err(err).Msg("Failed to create new Certificate Manager")
		}
	} else {
		var ca *x509.Certificate
		var caKey *rsa.PrivateKey
		if caPEM, caKeyPEM, ca, caKey, err = tresor.NewCA(*org, validityMinutes); err != nil {
			log.Fatal().Err(err).Msg("Failed to create new Certificate Authority")
		}
		certManager, err = tresor.NewCertManagerWithCA(ca, caKey, *org, validityMinutes)
	}

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to instantiate Certificate Manager")
	}

	cert, err := certManager.IssueCertificate(certificate.CommonName(*host))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to a new certificate")
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
		log.Fatal().Msgf("Invalid file name: %+v", fileName)
	}
	keyOut, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed open %s for write", fileName)
	}
	bytesWritten, err := keyOut.Write(content)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed writing content to file %s", fileName)
	}
	log.Info().Msgf("Wrote %d bytes to %s", bytesWritten, fileName)
	if err := keyOut.Close(); err != nil {
		log.Fatal().Err(err).Msgf("Error closing %s", fileName)
	}
}

func parseFlags() {
	if err := flags.Parse(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Error parsing command line arguments")
	}
	err := flag.CommandLine.Parse([]string{})
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing command line parameters")
	}
}
