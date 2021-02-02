package main

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/cli"
)

func TestCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI Test Suite")
}

var _ = BeforeSuite(func() {
	var err error
	chartTGZSource, err = cli.GetChartSource(filepath.Join("testdata", "test-chart"))
	Expect(err).NotTo(HaveOccurred())
})
