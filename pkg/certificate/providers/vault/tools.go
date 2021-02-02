package vault

import (
	"fmt"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
)

func getDurationInMinutes(validityPeriod time.Duration) string {
	return fmt.Sprintf("%dh", validityPeriod/time.Hour)
}

func getIssueURL(role vaultRole) vaultPath {
	return vaultPath(fmt.Sprintf("pki/issue/%+v", role))
}

func getRoleConfigURL(role vaultRole) vaultPath {
	return vaultPath(fmt.Sprintf("pki/roles/%s", role))
}

func getIssuanceData(cn certificate.CommonName, validityPeriod time.Duration) map[string]interface{} {
	return map[string]interface{}{
		commonNameField: cn.String(),
		ttlField:        getDurationInMinutes(validityPeriod),
	}
}
