package vault

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

var _ = Describe("Test tools", func() {
	Context("Test converting duration into Vault recognizable string", func() {
		It("converts 36 hours into correct string representation", func() {
			actual := getDurationInMinutes(2160 * time.Minute)
			expected := "36h"
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test cert issuance URL", func() {
		It("creates the URL for issuing a new certificate", func() {
			actual := getIssueURL()
			expected := "pki/issue/open-service-mesh"
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test role config URL", func() {
		It("creates the URL for role configuration", func() {
			actual := getRoleConfigURL()
			expected := "pki/roles/open-service-mesh"
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test cert issuance data for request", func() {
		It("creates a map w/ correct fields", func() {
			cn := certificate.CommonName("blah.foo.com")
			actual := getIssuanceData(cn, 8123*time.Minute)
			expected := map[string]interface{}{
				"common_name": "blah.foo.com",
				"ttl":         "135h",
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
