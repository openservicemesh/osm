package azure

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAzureProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure Endpoints Provider Suite")
}
