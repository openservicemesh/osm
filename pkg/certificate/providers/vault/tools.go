package vault

import (
	"fmt"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
)

func getDurationInMinutes(validityPeriod time.Duration) string {
	return fmt.Sprintf("%dh", validityPeriod/time.Hour)
}

func getIssueURL(role string) string {
	return fmt.Sprintf("pki/issue/%+v", role)
}

func getIssuanceData(options certificate.IssueOptions) map[string]interface{} {
	issuanceData := map[string]interface{}{
		commonNameField: options.CommonName().String(),
		ttlField:        getDurationInMinutes(options.ValidityDuration),
	}

	if options.URISAN().String() != "" {
		log.Trace().Str("cn", options.CommonName().String()).Msg("Generating Certificate with Uri SAN")

		// Comma delimited list; For SPIFFE compatibility it should only be one.
		issuanceData[uriSans] = options.URISAN().String()
	}

	return issuanceData
}
