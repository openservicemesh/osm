package dispatcher

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCatalog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dispatcher Test Suite")
}
