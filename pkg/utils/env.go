package utils

import (
	"fmt"
	"os"
)

// GetEnv is a convenience wrapper for os.Getenv() with additional default value return
// when empty or unset
func GetEnv(envVar string, defaultValue string) string {
	val := os.Getenv(envVar)
	if val == "" {
		return defaultValue
	}
	return val
}

// AppendClusterID is utilized to append extra DNS information to the FQDN if a clusterID is configured in the MeshConfig
// <svc-name>.<svc-namespace> becomes <svc-name>.<svc-namespace>.svc.cluster.<clusterID>
func AppendClusterID(baseFQDN, clusterID string) string {
	if clusterID == "" {
		return baseFQDN
	}
	return baseFQDN + fmt.Sprintf(".svc.cluster.%s", clusterID)
}
