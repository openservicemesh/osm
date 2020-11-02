package cli

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

var _ = Describe("New", func() {
	var flags *pflag.FlagSet

	BeforeEach(func() {
		flags = pflag.NewFlagSet("test-new", pflag.ContinueOnError)
	})

	It("sets the default namespace", func() {
		settings := New()
		settings.AddFlags(flags)
		flags.Parse(nil)
		Expect(settings.Namespace()).To(Equal(defaultOSMNamespace))
	})

	It("sets the namespace from the flag", func() {
		settings := New()
		settings.AddFlags(flags)
		err := flags.Parse([]string{"--osm-namespace=osm-ns"})
		Expect(err).To(BeNil())
		Expect(settings.Namespace()).To(Equal("osm-ns"))
	})

	It("sets the namespace from the env var", func() {
		os.Setenv(osmNamespaceEnvVar, "osm-env")
		defer os.Unsetenv(osmNamespaceEnvVar)

		settings := New()
		settings.AddFlags(flags)
		flags.Parse(nil)
		Expect(settings.Namespace()).To(Equal("osm-env"))
	})

	It("overrides the env var with the flag", func() {
		os.Setenv(osmNamespaceEnvVar, "osm-env")
		defer os.Unsetenv(osmNamespaceEnvVar)

		settings := New()
		settings.AddFlags(flags)
		err := flags.Parse([]string{"--osm-namespace=osm-ns"})
		Expect(err).To(BeNil())
		Expect(settings.Namespace()).To(Equal("osm-ns"))
	})
})
