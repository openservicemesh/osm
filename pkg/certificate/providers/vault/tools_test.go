package vault

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/certificate"
)

var _ = Describe("Test tools", func() {
	role := uuid.New().String()

	Context("Test converting duration into Vault recognizable string", func() {
		It("converts 36 hours into correct string representation", func() {
			actual := getDurationInMinutes(2160 * time.Minute)
			expected := "36h"
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test cert issuance URL", func() {
		It("creates the URL for issuing a new certificate", func() {
			actual := getIssueURL(role)
			expected := fmt.Sprintf("pki/issue/%s", role)
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test cert issuance data for request", func() {
		It("creates a map w/ correct fields", func() {
			options := certificate.NewCertOptionsWithFullName("blah.foo.com", 8123*time.Minute)
			actual := getIssuanceData(options)
			expected := map[string]interface{}{
				"common_name": "blah.foo.com",
				"ttl":         "135h",
			}
			Expect(actual).To(Equal(expected))
		})

		It("creates a map w/ correct fields when Spiffe Is enabled", func() {
			options := certificate.NewCertOptionsWithTrustDomain("sa.svc", "foo.com", 8123*time.Minute, true)
			actual := getIssuanceData(options)
			expected := map[string]interface{}{
				"common_name": "sa.svc.foo.com",
				"ttl":         "135h",
				"uri_sans":    "spiffe://foo.com/sa/svc",
			}
			Expect(actual).To(Equal(expected))
		})
	})

})
