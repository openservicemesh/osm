package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMaestro(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Maestro Test Suite")
}
