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

func getIssuanceData(cn certificate.CommonName, validityPeriod time.Duration) map[string]interface{} {
	return map[string]interface{}{
		commonNameField: cn.String(),
		ttlField:        getDurationInMinutes(validityPeriod),
	}
}
