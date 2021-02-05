package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/injector"
)

var _ = Describe("Test validateCertificateManagerOptions", func() {
	var (
		testCaBundleSecretName = "test-secret"
	)

	Context("tresor certProviderKind is passed in", func() {
		certProviderKind = providers.TresorKind.String()

		err := validateCertificateManagerOptions()

		It("should not error", func() {
			Expect(err).To(BeNil())
		})
	})
	Context("vault certProviderKind is passed in and vaultToken is not empty", func() {
		certProviderKind = providers.VaultKind.String()
		vaultOptions.VaultToken = "anythinghere"

		err := validateCertificateManagerOptions()

		It("should not error", func() {
			Expect(err).To(BeNil())
		})
	})
	Context("vault certProviderKind is passed in but vaultToken is empty", func() {
		certProviderKind = providers.VaultKind.String()
		vaultOptions.VaultToken = ""

		err := validateCertificateManagerOptions()

		It("should error", func() {
			Expect(err).To(HaveOccurred())

		})
	})
	Context("cert-manager certProviderKind is passed in with valid caBundleSecretName and certmanagerIssuerName", func() {
		certProviderKind = providers.CertManagerKind.String()
		caBundleSecretName = testCaBundleSecretName
		certManagerOptions.IssuerName = "test-issuer"

		err := validateCertificateManagerOptions()

		It("should not error", func() {
			Expect(err).To(BeNil())
		})
	})
	Context("cert-manager certProviderKind is passed in with caBundleSecretName but no certmanagerIssureName", func() {
		certProviderKind = providers.CertManagerKind.String()
		caBundleSecretName = testCaBundleSecretName
		certManagerOptions.IssuerName = ""

		err := validateCertificateManagerOptions()

		It("should error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
	Context("cert-manager certProviderKind is passed in without caBundleSecretName but no certmanagerIssureName", func() {
		certProviderKind = providers.CertManagerKind.String()
		caBundleSecretName = ""
		certManagerOptions.IssuerName = ""

		err := validateCertificateManagerOptions()

		It("should error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	Context("invalid kind is passed in", func() {
		certProviderKind = "invalidkind"

		err := validateCertificateManagerOptions()

		It("should error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Test validateCLIParams", func() {
	var (
		testMeshName           = "test-mesh-name"
		testOsmNamespace       = "test-namespace"
		testInitContainerImage = "test-init-image"
		testSidecarImage       = "test-sidecar-image"
		testwebhookConfigName  = "test-webhook-name"
		testCABundleSecretName = "test-ca-bundle"
	)

	Context("none of the necessary CLI params are empty", func() {
		certProviderKind = providers.TresorKind.String()
		meshName = testMeshName
		osmNamespace = testOsmNamespace
		injectorConfig = injector.Config{
			InitContainerImage: testInitContainerImage,
			SidecarImage:       testSidecarImage,
		}
		webhookConfigName = testwebhookConfigName
		caBundleSecretName = testCABundleSecretName

		err := validateCLIParams()

		It("should not error", func() {
			Expect(err).To(BeNil())
		})
	})
	Context("mesh name is empty", func() {
		certProviderKind = providers.TresorKind.String()
		meshName = ""
		osmNamespace = testOsmNamespace
		injectorConfig = injector.Config{
			InitContainerImage: testInitContainerImage,
			SidecarImage:       testSidecarImage,
		}
		webhookConfigName = testwebhookConfigName

		err := validateCLIParams()

		It("should error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
	Context("osmNamespace is empty", func() {
		certProviderKind = providers.TresorKind.String()
		meshName = testMeshName
		osmNamespace = ""
		injectorConfig = injector.Config{
			InitContainerImage: testInitContainerImage,
			SidecarImage:       testSidecarImage,
		}
		webhookConfigName = testwebhookConfigName

		err := validateCLIParams()

		It("should error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
	Context("InitContainerImage on injectorConfig is empty", func() {
		certProviderKind = providers.TresorKind.String()
		meshName = testMeshName
		osmNamespace = testOsmNamespace
		injectorConfig = injector.Config{
			InitContainerImage: "",
			SidecarImage:       testSidecarImage,
		}
		webhookConfigName = testwebhookConfigName

		err := validateCLIParams()

		It("should error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
	Context("SidecarImage on injectorConfig is empty", func() {
		certProviderKind = providers.TresorKind.String()
		meshName = testMeshName
		osmNamespace = testOsmNamespace
		injectorConfig = injector.Config{
			InitContainerImage: testInitContainerImage,
			SidecarImage:       "",
		}
		webhookConfigName = testwebhookConfigName

		err := validateCLIParams()

		It("should error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
	Context("webhookConfigName is empty", func() {
		certProviderKind = providers.TresorKind.String()
		meshName = testMeshName
		osmNamespace = testOsmNamespace
		injectorConfig = injector.Config{
			InitContainerImage: testInitContainerImage,
			SidecarImage:       testSidecarImage,
		}
		webhookConfigName = ""

		err := validateCLIParams()

		It("should error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
})
