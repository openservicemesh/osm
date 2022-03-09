package rotor_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/rotor"
	"github.com/openservicemesh/osm/pkg/messaging"
)

type fakeIssuer struct{}

func (i *fakeIssuer) IssueCertificate(cn certificate.CommonName, validityPeriod time.Duration) (*certificate.Certificate, error) {
	return &certificate.Certificate{
		CommonName: cn,
		Expiration: time.Now().Add(validityPeriod),
	}, nil
}

var _ = Describe("Test Rotor", func() {

	cn := certificate.CommonName("foo")

	Context("Testing rotating expiring certificates", func() {

		validityPeriod := -1 * time.Hour // negative time means this cert has already expired -- will be rotated asap

		stop := make(chan struct{})
		defer close(stop)
		msgBroker := messaging.NewBroker(stop)
		certManager, err := certificate.NewManager(&certificate.Certificate{}, &fakeIssuer{}, validityPeriod, msgBroker)

		Expect(err).ToNot(HaveOccurred())

		certA, err := certManager.IssueCertificate(cn, validityPeriod)

		certRotateChan := msgBroker.GetCertPubSub().Sub(announcements.CertificateRotated.String())

		It("issued a new certificate", func() {
			Expect(err).ToNot(HaveOccurred())
		})

		It("rotates certificate", func() {
			done := make(chan interface{})

			start := time.Now()
			rotor.New(certManager).Start(360 * time.Second)
			// Wait for one certificate rotation to be announced and terminate
			<-certRotateChan
			close(done)

			fmt.Printf("It took %+v to rotate certificate %s\n", time.Since(start), cn)

			newCert, err := certManager.IssueCertificate(cn, validityPeriod)
			Expect(err).ToNot(HaveOccurred())
			Expect(newCert.GetExpiration()).ToNot(Equal(certA.GetExpiration()))
			Expect(newCert).ToNot(Equal(certA))
		})
	})

})
