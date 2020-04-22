package main

import (
	"flag"
	"os"
	"time"

	"github.com/spf13/pflag"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/logger"
	"github.com/open-service-mesh/osm/pkg/tresor"
)

var (
	flags           = pflag.NewFlagSet("certificate-tresor", pflag.ExitOnError)
	host            = flags.String("host", "bookstore.mesh", "host name for the certificate")
	out             = flags.String("out", "", "full path to the certificate PEM file")
	keyout          = flags.String("keyout", "", "full path to the private key PEM file")
	caPEMFileIn     = flags.String("caPEMFileIn", "", "full path to the root cert to be loaded")
	caKeyPEMFileIn  = flags.String("caKeyPEMFileIn", "", "full path to the root cert key to be loaded")
	caPEMFileOut    = flags.String("caPEMFileOut", "", "full path to the root cert to be created")
	caKeyPEMFileOut = flags.String("caKeyPEMFileOut", "", "full path to the root cert key to be created")
	validity        = flags.Int("validity", 525600, "validity duration of a certificate in MINUTES")
	genca           = flags.Bool("genca", false, "set this flag to true to generate a new CA certificate; no other certificates will be created")
)

var (
	log = logger.New("cert")
)

func main() {
	parseFlags()

	if *genca {
		validityMinutes := time.Duration(*validity) * time.Minute
		rootCert, err := tresor.NewCA(validityMinutes)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create new Certificate Authority")
		}
		writeFile(*caPEMFileOut, rootCert.GetCertificateChain())
		writeFile(*caKeyPEMFileOut, rootCert.GetPrivateKey())
		os.Exit(0)
	}

	certManager, rootCert := getCertManager()

	if *caPEMFileOut != "" {
		writeFile(*caPEMFileOut, rootCert.GetCertificateChain())
	}
	if *caKeyPEMFileOut != "" {
		writeFile(*caKeyPEMFileOut, rootCert.GetPrivateKey())
	}

	cert, err := certManager.IssueCertificate(certificate.CommonName(*host))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to a new certificate")
	}

	writeFile(*out, cert.GetCertificateChain())
	writeFile(*keyout, cert.GetPrivateKey())

}

func getCertManager() (*tresor.CertManager, *tresor.Certificate) {
	validityMinutes := time.Duration(*validity) * time.Minute

	if caPEMFileIn != nil && caKeyPEMFileIn != nil && *caPEMFileIn != "" && *caKeyPEMFileIn != "" {
		ca, err := tresor.LoadCA(*caPEMFileIn, *caKeyPEMFileIn)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error loading root certificate & key from files %s and %s", *caPEMFileIn, *caKeyPEMFileIn)
		}
		certManager, err := tresor.NewCertManager(ca, validityMinutes)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create new Certificate Manager")
		}
		return certManager, ca
	}

	ca, err := tresor.NewCA(validityMinutes)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create new Certificate Authority")
	}
	certManager, err := tresor.NewCertManager(ca, validityMinutes)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to instantiate Certificate Manager")
	}
	return certManager, ca
}

func writeFile(fileName string, content []byte) {
	if fileName == "" {
		log.Error().Msgf("Invalid file name: %+v", fileName)
		return
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
