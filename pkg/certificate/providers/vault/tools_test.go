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
	role := vaultRole(uuid.New().String())

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
			expected := vaultPath(fmt.Sprintf("pki/issue/%s", role))
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test role config URL", func() {
		It("creates the URL for role configuration", func() {
			actual := getRoleConfigURL(role)
			expected := vaultPath(fmt.Sprintf("pki/roles/%s", role))
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
