package cli

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chartutil"
)

var _ = Describe("Chart Source", func() {
	It("Loads a chart without errors", func() {
		By("Creating a temp dir")

		tmp, err := ioutil.TempDir(os.TempDir(), "osm-test")
		Expect(err).NotTo(HaveOccurred())

		defer func() {
			By("cleaning up the temp dir")
			err := os.RemoveAll(tmp)
			if err != nil {
				GinkgoT().Log(err)
			}
		}()

		By("Creating a dummy chart")

		chartName := "test-chart"
		chartPath, err := chartutil.Create(chartName, tmp)
		Expect(err).NotTo(HaveOccurred())

		By("Getting source from the chart path")

		source, err := GetChartSource(chartPath)
		Expect(err).NotTo(HaveOccurred())

		By("Loading the chart source")

		ch, err := LoadChart(source)
		Expect(err).NotTo(HaveOccurred())

		Expect(ch.Name()).To(Equal(chartName))
	})
})
