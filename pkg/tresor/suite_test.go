package tresor

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTresor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tresor Test Suite")
}
