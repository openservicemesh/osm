package main

import (
	"crypto/rsa"
	"crypto/x509"
	goflag "flag"
	"os"
	"time"

	"github.com/golang/glog"
	flag "github.com/spf13/pflag"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/tresor"
	"github.com/open-service-mesh/osm/pkg/tresor/pem"
)

var (
	host            string
	out             string
	keyout          string
	caPEMFileIn     string
	caKeyPEMFileIn  string
	caPEMFileOut    string
	caKeyPEMFileOut string
	org             string
	validity        int
)

func init() {
	flag.StringVar(&host, "host", "bookstore.mesh", "host name for the certificate")
	flag.StringVar(&out, "out", "", "full path to the certificate PEM file")
	flag.StringVar(&keyout, "keyout", "", "full path to the private key PEM file")
	flag.StringVar(&caPEMFileIn, "caPEMFileIn", "", "full path to the root cert to be loaded")
	flag.StringVar(&caKeyPEMFileIn, "caKeyPEMFileIn", "", "full path to the root cert key to be loaded")
	flag.StringVar(&caPEMFileOut, "caPEMFileOut", "", "full path to the root key to be created")
	flag.StringVar(&caKeyPEMFileOut, "caKeyPEMFileOut", "", "full path to the root cert key to be created")
	flag.StringVar(&org, "org", "ACME Co", "name of the organization for certificate manager")
	flag.IntVar(&validity, "validity", 525600, "validity duration of a certificate in MINUTES")
}

func main() {
	defer glog.Flush()
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	var caPEM pem.RootCertificate
	var caKeyPEM pem.RootPrivateKey
	var certManager *tresor.CertManager
	var err error

	validityMinutes := time.Duration(validity) * time.Minute

	if caPEMFileIn != "" && caKeyPEMFileIn != "" {
		if certManager, err = tresor.NewCertManagerWithCAFromFile(caPEMFileIn, caKeyPEMFileIn, org, validityMinutes); err != nil {
			glog.Fatal(err)
		}
	} else {
		var ca *x509.Certificate
		var caKey *rsa.PrivateKey
		if caPEM, caKeyPEM, ca, caKey, err = tresor.NewCA(org, validityMinutes); err != nil {
			glog.Fatal(err)
		}
		certManager, err = tresor.NewCertManagerWithCA(ca, caKey, org, validityMinutes)
	}

	if err != nil {
		glog.Fatal("Could not instantiate Certificate Manager: ", err)
	}

	cert, err := certManager.IssueCertificate(certificate.CommonName(host))
	if err != nil {
		glog.Fatal("Error creating a new certificate: ", err)
	}

	if caPEMFileOut != "" {
		writeFile(caPEMFileOut, caPEM)
	}
	if caKeyPEMFileOut != "" {
		writeFile(caKeyPEMFileOut, caKeyPEM)
	}

	if out != "" {
		writeFile(out, cert.GetCertificateChain())
	}
	if keyout != "" {
		writeFile(keyout, cert.GetPrivateKey())
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
