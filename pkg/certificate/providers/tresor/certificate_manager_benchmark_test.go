package tresor

import (
	"testing"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/logger"
)

var bmCert *certificate.Certificate

func BenchmarkIssueCertificate(b *testing.B) {
	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	serviceFQDN := certificate.CommonName("a.b.c")
	validity := 3 * time.Second
	cn := certificate.CommonName("Test CA")
	rootCertCountry := "US"
	rootCertLocality := "CA"
	rootCertOrganization := testCertOrgName

	rootCert, err := NewCA(cn, 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		b.Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err.Error())
	}

	m, newCertError := New(
		rootCert,
		"org",
		2048,
	)
	if newCertError != nil {
		b.Fatalf("Error creating new certificate manager: %s", newCertError.Error())
	}

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		bmCert, _ = m.IssueCertificate(serviceFQDN, validity)
	}
	b.StopTimer()
}
