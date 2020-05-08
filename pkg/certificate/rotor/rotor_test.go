package rotor_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/certificate/rotor"
)

var _ = Describe("Test Rotisserie", func() {

	cn := certificate.CommonName("foo")

	Context("Testing rotating expiring certificates", func() {
		cache := make(map[certificate.CommonName]certificate.Certificater)
		validityPeriod := 1 * time.Hour
		certManager := tresor.NewFakeCertManager(&cache, validityPeriod)

		It("determines whether a certificate has expired", func() {
			cert, err := certManager.IssueCertificate(cn)
			Expect(err).ToNot(HaveOccurred())
			actual := rotor.ShouldRotate(cert)
			Expect(actual).To(BeFalse())
		})
	})

	Context("Testing rotating expiring certificates", func() {
		cache := make(map[certificate.CommonName]certificate.Certificater)
		validityPeriod := -1 * time.Hour // negative time means this cert has already expired -- will be rotated asap
		certManager := tresor.NewFakeCertManager(&cache, validityPeriod)

		certA, err := certManager.IssueCertificate(cn)

		It("issued a new certificate", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(certA).To(Equal(cache[cn]))
		})

		It("will determine that the certificate needs to be rotated because it has already expired due to negative validity period", func() {
			actual := rotor.ShouldRotate(certA)
			Expect(actual).To(BeTrue())
		})

		It("rotates certificate", func() {
			done := make(chan interface{})
			Expect(cache[cn]).To(Equal(certA))

			start := time.Now()
			announcements := rotor.New(1*time.Millisecond, done, certManager, &cache)
			// Wait for one certificate rotation to be announced and terminate
			<-announcements
			close(done)

			fmt.Printf("It took %+v to rotate certificate %s\n", time.Since(start), cn)

			newCert, err := certManager.IssueCertificate(cn)
			Expect(err).ToNot(HaveOccurred())
			Expect(newCert.GetExpiration()).ToNot(Equal(certA.GetExpiration()))
			Expect(newCert).ToNot(Equal(certA))
		})
	})

})
