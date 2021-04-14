package main

import (
	"bytes"
	"context"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	meshName = "testing"
)

var _ = Describe("Running the uninstall command", func() {
	Context("default parameters", func() {
		var (
			cmd   *uninstallCmd
			force bool
		)

		When("the mesh exists", func() {
			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: meshName})
			err := store.Create(rel)
			Expect(err).To(BeNil())

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
			cmd = &uninstallCmd{
				out:      out,
				in:       in,
				client:   helm.NewUninstall(testConfig),
				meshName: meshName,
				force:    force,
			}

			err = cmd.run()

			It("should prompt for confirmation", func() {
				Expect(out.String()).To(ContainSubstring("Uninstall OSM [mesh name: testing] ? [y/n]: "))
			})
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should give a message confirming the successful uninstall", func() {
				Expect(out.String()).To(ContainSubstring("OSM [mesh name: testing] uninstalled\n"))
			})

		})

		When("the mesh doesn't exist", func() {
			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: "other-mesh"})
			err := store.Create(rel)
			Expect(err).To(BeNil())

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
			cmd = &uninstallCmd{
				out:      out,
				in:       in,
				client:   helm.NewUninstall(testConfig),
				meshName: meshName,
				force:    force,
			}

			err = cmd.run()

			It("should prompt for confirmation", func() {
				Expect(out.String()).To(ContainSubstring("Uninstall OSM [mesh name: testing] ? [y/n]: "))
			})
			It("should error", func() {
				Expect(err).To(MatchError("No OSM control plane with mesh name [testing] found in namespace [osm-system]"))
			})
			It("should not give a message confirming the successful uninstall", func() {
				Expect(out.String()).ToNot(ContainSubstring("OSM [mesh name: testing] uninstalled\n"))
			})

		})
	})
	Context("custom parameters", func() {
		var (
			cmd             *uninstallCmd
			force           bool
			deleteNamespace bool
		)
		When("force is true", func() {
			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: "other-mesh"})
			err := store.Create(rel)
			Expect(err).To(BeNil())

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
			cmd = &uninstallCmd{
				out:      out,
				in:       in,
				client:   helm.NewUninstall(testConfig),
				meshName: meshName,
				force:    force,
			}

			err = cmd.run()

			It("should not prompt for confirmation", func() {
				Expect(out.String()).NotTo(ContainSubstring("Uninstall OSM [mesh name: testing] ? [y/n]: "))
			})
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not give a message confirming the uninstall", func() {
				Expect(out.String()).ToNot(ContainSubstring("OSM [mesh name: testing] uninstalled\n"))
			})

		})

		When("delete-namespace is true, but user enters no when prompted", func() {
			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: "other-mesh"})
			err := store.Create(rel)
			Expect(err).To(BeNil())

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet := fake.NewSimpleClientset()
			out := new(bytes.Buffer)
			in := new(bytes.Buffer)
			in.Write([]byte("n\n"))
			deleteNamespace = true
			cmd = &uninstallCmd{
				out:             out,
				in:              in,
				client:          helm.NewUninstall(testConfig),
				meshName:        meshName,
				deleteNamespace: deleteNamespace,
				clientSet:       fakeClientSet,
			}

			err = cmd.run()

			It("should prompt for confirmation", func() {
				Expect(out.String()).To(ContainSubstring("Uninstall OSM [mesh name: testing] ? [y/n]: "))
			})
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not give a message confirming the uninstall of OSM", func() {
				Expect(out.String()).ToNot(ContainSubstring("OSM [mesh name: " + meshName + "] uninstalled\n"))
			})
			It("should not give a message confirming the deletion on namespace", func() {
				Expect(out.String()).ToNot(ContainSubstring("OSM namespace [" + settings.Namespace() + "] deleted successfully\n"))
			})

		})

		When("delete-namespace is true and user enters yes when prompted", func() {
			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: meshName})
			err := store.Create(rel)
			Expect(err).To(BeNil())

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			// create namespace so that it can be deleted
			fakeClientSet := fake.NewSimpleClientset()
			_, err = fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), createNamespaceSpec(settings.Namespace(),
				meshName, true), v1.CreateOptions{})
			Expect(err).To(BeNil())

			out := new(bytes.Buffer)
			in := new(bytes.Buffer)
			in.Write([]byte("y\n"))
			deleteNamespace = true
			cmd = &uninstallCmd{
				out:             out,
				in:              in,
				client:          helm.NewUninstall(testConfig),
				meshName:        meshName,
				deleteNamespace: deleteNamespace,
				clientSet:       fakeClientSet,
			}

			err = cmd.run()

			It("should prompt for confirmation", func() {
				Expect(out.String()).To(ContainSubstring("Uninstall OSM [mesh name: testing] ? [y/n]: "))
			})
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should give a message confirming the uninstall of OSM", func() {
				Expect(out.String()).To(ContainSubstring("OSM [mesh name: " + meshName + "] uninstalled\n"))
			})
			It("should give a message confirming the deletion on namespace", func() {
				Expect(out.String()).To(ContainSubstring("OSM namespace [" + settings.Namespace() + "] deleted successfully\n"))
			})
		})
	})
})
