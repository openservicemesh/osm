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
			meshName  string
			force     bool
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

			out := new(bytes.Buffer)
			in := new(bytes.Buffer)
			in.Write([]byte("y\n"))
			force = false
			deleteCmd = &meshDeleteCmd{
				out:      out,
				in:       in,
				client:   helm.NewUninstall(testConfig),
				meshName: meshName,
				force:    force,
			}

			err := deleteCmd.run()

			It("should prompt for confirmation", func() {
				Expect(out.String()).To(ContainSubstring("Delete OSM [mesh name: testing] ? [y/n]: "))
			})
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should give a message confirming the successful delete", func() {
				Expect(out.String()).To(ContainSubstring("OSM [mesh name: testing] deleted\n"))
			})

		})

		When("the mesh doesn't exist", func() {
			meshName = "testing"
			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: "other-mesh"})
			store.Create(rel)

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			out := new(bytes.Buffer)
			in := new(bytes.Buffer)
			in.Write([]byte("y\n"))
			force = false
			deleteCmd = &meshDeleteCmd{
				out:      out,
				in:       in,
				client:   helm.NewUninstall(testConfig),
				meshName: meshName,
				force:    force,
			}

			err := deleteCmd.run()

			It("should prompt for confirmation", func() {
				Expect(out.String()).To(ContainSubstring("Delete OSM [mesh name: testing] ? [y/n]: "))
			})
			It("should error", func() {
				Expect(err).To(MatchError("No OSM control plane with mesh name [testing] found in namespace [osm-system]"))
			})
			It("should not give a message confirming the successful delete", func() {
				Expect(out.String()).ToNot(ContainSubstring("OSM [mesh name: testing] deleted\n"))
			})

		})

		When("force is true", func() {
			meshName = "testing"
			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: "other-mesh"})
			store.Create(rel)

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			out := new(bytes.Buffer)
			in := new(bytes.Buffer)
			force = true
			deleteCmd = &meshDeleteCmd{
				out:      out,
				in:       in,
				client:   helm.NewUninstall(testConfig),
				meshName: meshName,
				force:    force,
			}

			err := deleteCmd.run()

			It("should not prompt for confirmation", func() {
				Expect(out.String()).NotTo(ContainSubstring("Delete OSM [mesh name: testing] ? [y/n]: "))
			})
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not give a message confirming the delete", func() {
				Expect(out.String()).ToNot(ContainSubstring("OSM [mesh name: testing] deleted\n"))
			})

		})
	})
})
