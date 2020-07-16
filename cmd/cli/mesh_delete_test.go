package main

import (
	"bytes"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

var _ = Describe("Running the mesh delete command", func() {
	Context("default parameters", func() {
		var (
			deleteCmd *meshDeleteCmd
			out       *bytes.Buffer
			meshName  string
		)

		When("the mesh exists", func() {
			meshName = "testing"
			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: meshName})
			store.Create(rel)

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			out = new(bytes.Buffer)
			deleteCmd = &meshDeleteCmd{
				out:    out,
				client: helm.NewUninstall(testConfig),
				name:   meshName,
			}

			err := deleteCmd.run()

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should give a message confirming the successful install", func() {
				Expect(out.String()).To(Equal("OSM [mesh name: testing] deleted\n"))
			})

		})
	})

})
