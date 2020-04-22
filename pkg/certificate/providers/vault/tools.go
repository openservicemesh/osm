package vault

import (
	"fmt"
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

func getDurationInMinutes(validity time.Duration) string {
	return fmt.Sprintf("%dh", validity/time.Hour)
}

func getIssueURL() string {
	return fmt.Sprintf("pki/issue/%+v", vaultRole)
}

func getRoleConfigURL() string {
	return fmt.Sprintf("pki/roles/%s", vaultRole)
}

func getIssuanceData(cn certificate.CommonName, validity time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"common_name": cn.String(),
		"ttl":         getDurationInMinutes(validity),
	}
}
