package debugger

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
)

func (ds DebugConfig) getCertHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		certs := ds.certDebugger.ListIssuedCertificates()

		sort.Slice(certs, func(i, j int) bool {
			return certs[i].GetCommonName() < certs[j].GetCommonName()
		})

		for idx, cert := range certs {
			ca := cert.GetIssuingCA()
			chain := cert.GetCertificateChain()
			x509, err := certificate.DecodePEMCertificate(chain)
			if err != nil {
				log.Error().Err(err).Msgf("Error decoding PEM to x509 SerialNumber=%s", cert.GetSerialNumber())
			}

			_, _ = fmt.Fprintf(w, "---[ %d ]---\n", idx)
			_, _ = fmt.Fprintf(w, "\t Common Name: %q\n", cert.GetCommonName())
			_, _ = fmt.Fprintf(w, "\t Valid Until: %+v (%+v remaining)\n", cert.GetExpiration(), time.Until(cert.GetExpiration()))
			_, _ = fmt.Fprintf(w, "\t Issuing CA (SHA256): %x\n", sha256.Sum256(ca))
			_, _ = fmt.Fprintf(w, "\t Cert Chain (SHA256): %x\n", sha256.Sum256(chain))

			// Show only some x509 fields to keep the output clean
			_, _ = fmt.Fprintf(w, "\t x509.SignatureAlgorithm: %+v\n", x509.SignatureAlgorithm)
			_, _ = fmt.Fprintf(w, "\t x509.PublicKeyAlgorithm: %+v\n", x509.PublicKeyAlgorithm)
			_, _ = fmt.Fprintf(w, "\t x509.Version: %+v\n", x509.Version)
			_, _ = fmt.Fprintf(w, "\t x509.SerialNumber: %x\n", x509.SerialNumber)
			_, _ = fmt.Fprintf(w, "\t x509.Issuer: %+v\n", x509.Issuer)
			_, _ = fmt.Fprintf(w, "\t x509.Subject: %+v\n", x509.Subject)
			_, _ = fmt.Fprintf(w, "\t x509.NotBefore (begin): %+v (%+v ago)\n", x509.NotBefore, time.Since(x509.NotBefore))
			_, _ = fmt.Fprintf(w, "\t x509.NotAfter (end): %+v (%+v remaining)\n", x509.NotAfter, time.Until(x509.NotAfter))
			_, _ = fmt.Fprintf(w, "\t x509.BasicConstraintsValid: %+v\n", x509.BasicConstraintsValid)
			_, _ = fmt.Fprintf(w, "\t x509.IsCA: %+v\n", x509.IsCA)
			_, _ = fmt.Fprintf(w, "\t x509.DNSNames: %+v\n", x509.DNSNames)

			_, _ = fmt.Fprintf(w, "\t Cert struct expiration vs. x509.NotAfter: %+v\n", x509.NotAfter.Sub(cert.GetExpiration()))

			_, _ = fmt.Fprint(w, "\n")
		}
	})
}
