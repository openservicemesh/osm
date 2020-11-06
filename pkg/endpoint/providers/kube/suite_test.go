package kube

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestKubeProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kube Provider Test Suite")
}
