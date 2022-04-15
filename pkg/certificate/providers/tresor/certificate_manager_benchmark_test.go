package tresor

import (
	"testing"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
)


func BenchmarkIssueCertificate(b *testing.B) {
	serviceFQDN := certificate.CommonName("a.b.c")
	validity := 3 * time.Second
	cn := certificate.CommonName("Test CA")
	rootCertCountry := "US"
	rootCertLocality := "CA"
	rootCertOrganization := "Open Service Mesh Tresor"

	rootCert, err := NewCA(cn, 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		b.Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err.Error())
	}

	for i := 0; i < b.N; i++ {
		m, newCertError := New(
			rootCert,
			"org",
			2048,
		)
		if newCertError != nil {
			b.Fatalf("Error creating new certificate manager: %s", newCertError.Error())
		}
		m.IssueCertificate(serviceFQDN, validity)
	}
}