package rotor_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/certificate/rotor"
)

var _ = Describe("Test Rotor", func() {

	cn := certificate.CommonName("foo")

	Context("Testing rotating expiring certificates", func() {
		cache := make(map[certificate.CommonName]certificate.Certificater)
		validityPeriod := 1 * time.Hour
		certManager := tresor.NewFakeCertManager(&cache, validityPeriod)

		It("determines whether a certificate has expired", func() {
			cert, err := certManager.IssueCertificate(cn, nil)
			Expect(err).ToNot(HaveOccurred())
			actual := rotor.ShouldRotate(cert)
			Expect(actual).To(BeFalse())
		})
	})

	Context("Testing rotating expiring certificates", func() {
		cache := make(map[certificate.CommonName]certificate.Certificater)
		validityPeriod := -1 * time.Hour // negative time means this cert has already expired -- will be rotated asap
		certManager := tresor.NewFakeCertManager(&cache, validityPeriod)

		certA, err := certManager.IssueCertificate(cn, nil)

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
			rotor.New(certManager).Start(360 * time.Second)
			// Wait for one certificate rotation to be announced and terminate
			<-certManager.GetAnnouncementsChannel()
			close(done)

			fmt.Printf("It took %+v to rotate certificate %s\n", time.Since(start), cn)

			newCert, err := certManager.IssueCertificate(cn, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(newCert.GetExpiration()).ToNot(Equal(certA.GetExpiration()))
			Expect(newCert).ToNot(Equal(certA))
		})
	})

})
