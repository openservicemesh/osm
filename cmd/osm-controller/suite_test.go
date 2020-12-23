package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestADSMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ADSMain Test Suite")
}
