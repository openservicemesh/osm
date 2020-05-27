package vault

import (
	"fmt"
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

func getDurationInMinutes(validityPeriod time.Duration) string {
	return fmt.Sprintf("%dh", validityPeriod/time.Hour)
}

func getIssueURL(vaultRole string) string {
	return fmt.Sprintf("pki/issue/%+v", vaultRole)
}

func getRoleConfigURL(vaultRole string) string {
	return fmt.Sprintf("pki/roles/%s", vaultRole)
}

func getIssuanceData(cn certificate.CommonName, validityPeriod time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"common_name": cn.String(),
		"ttl":         getDurationInMinutes(validityPeriod),
	}
}
