package cli

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCLIUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Suite")
}

var oldEnv *string

var _ = BeforeSuite(func() {
	if e, ok := os.LookupEnv(osmNamespaceEnvVar); ok {
		oldEnv = new(string)
		*oldEnv = e
		os.Unsetenv(osmNamespaceEnvVar)
	}
})

var _ = AfterSuite(func() {
	if oldEnv != nil {
		os.Setenv(osmNamespaceEnvVar, *oldEnv)
	}
})
