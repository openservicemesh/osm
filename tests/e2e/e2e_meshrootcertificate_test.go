package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	certman "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	. "github.com/openservicemesh/osm/tests/framework"
)

var (
	ActiveIntent  = v1alpha2.MeshRootCertificateIntent("active")
	PassiveIntent = v1alpha2.MeshRootCertificateIntent("passive")
)

var _ = OSMDescribe("MeshRootCertificate",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 11,
	},
	func() {
		Context("with Tressor", func() {
			It("rotates certificates", func() {
				basicCertRotationScenario()
			})
		})

		Context("with CertManager", func() {
			It("rotates certificates", func() {
				basicCertRotationScenario(WithCertManagerEnabled())
			})
		})

		Context("with Vault", func() {
			It("rotates certificates", func() {
				basicCertRotationScenario(WithVault())
			})
		})
	})

func basicCertRotationScenario(installOptions ...InstallOsmOpt) {
	By("installing with MRC enabled")
	installOptions = append(installOptions, WithMeshRootCertificateEnabled())
	installOpts := Td.GetOSMInstallOpts(installOptions...)
	Expect(Td.InstallOSM(installOpts)).To(Succeed())

	// no secrets are created in Vault case
	if installOpts.CertManager != Vault {
		By("checking the certificate exists")
		err := Td.WaitForCABundleSecret(Td.OsmNamespace, OsmCABundleName, time.Second*5)
		Expect(err).NotTo(HaveOccurred())
	}

	By("checking that an active cert cannot be created")
	time.Sleep(time.Second * 10)
	activeNotAllowed := "not-allowed"
	_, err := createMeshRootCertificate(activeNotAllowed, ActiveIntent, installOpts.CertManager)
	Expect(err).Should(HaveOccurred())
	Expect(err.Error()).Should(ContainSubstring("cannot create MRC %s/%s with intent active. An MRC with active intent already exists in the control plane namespace", Td.OsmNamespace, activeNotAllowed))

	By("creating a second certificate in passive state")
	newCertName := "osm-mrc-2"
	_, err = createMeshRootCertificate(newCertName, PassiveIntent, installOpts.CertManager)
	Expect(err).NotTo(HaveOccurred())

	// no secrets are created in Vault case
	if installOpts.CertManager != Vault {
		By("ensuring the new CA secret exists")
		err = Td.WaitForCABundleSecret(Td.OsmNamespace, newCertName, time.Second*90)
		Expect(err).NotTo(HaveOccurred())
	}
	// TODO(#4835) add checks for the correct statuses for the two certificates and complete cert rotation
}

func createMeshRootCertificate(name string, intent v1alpha2.MeshRootCertificateIntent, certificateManagerType string) (*v1alpha2.MeshRootCertificate, error) {
	switch certificateManagerType {
	case DefaultCertManager:
		return createTressorMRC(name, intent)
	case CertManager:
		return createCertManagerMRC(name, intent)
	case Vault:
		return createVaultMRC(name, intent)
	default:
		Fail("should not be able to create MRC of unknown type")
		return nil, fmt.Errorf("should not be able to create MRC of unknown type")
	}
}

func createTressorMRC(name string, intent v1alpha2.MeshRootCertificateIntent) (*v1alpha2.MeshRootCertificate, error) {
	return Td.ConfigClient.ConfigV1alpha2().MeshRootCertificates(Td.OsmNamespace).Create(
		context.Background(), &v1alpha2.MeshRootCertificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: Td.OsmNamespace,
			},
			Spec: v1alpha2.MeshRootCertificateSpec{
				TrustDomain: "cluster.local",
				Intent:      intent,
				Provider: v1alpha2.ProviderSpec{
					Tresor: &v1alpha2.TresorProviderSpec{
						CA: v1alpha2.TresorCASpec{
							SecretRef: v1.SecretReference{
								Name:      name,
								Namespace: Td.OsmNamespace,
							},
						}},
				},
			},
		}, metav1.CreateOptions{})
}

func createCertManagerMRC(name string, intent v1alpha2.MeshRootCertificateIntent) (*v1alpha2.MeshRootCertificate, error) {
	cert := &cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: cmapi.CertificateSpec{
			IsCA:       true,
			Duration:   &metav1.Duration{Duration: 90 * 24 * time.Hour},
			SecretName: name,
			CommonName: "osm-system",
			IssuerRef: cmmeta.ObjectReference{
				Name:  "selfsigned",
				Kind:  "Issuer",
				Group: "cert-manager.io",
			},
		},
	}

	ca := &cmapi.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: cmapi.IssuerSpec{
			IssuerConfig: cmapi.IssuerConfig{
				CA: &cmapi.CAIssuer{
					SecretName: name,
				},
			},
		},
	}

	cmClient, err := certman.NewForConfig(Td.RestConfig)
	Expect(err).NotTo(HaveOccurred())

	_, err = cmClient.CertmanagerV1().Certificates(Td.OsmNamespace).Create(context.TODO(), cert, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	_, err = cmClient.CertmanagerV1().Issuers(Td.OsmNamespace).Create(context.TODO(), ca, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	return Td.ConfigClient.ConfigV1alpha2().MeshRootCertificates(Td.OsmNamespace).Create(
		context.Background(), &v1alpha2.MeshRootCertificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: Td.OsmNamespace,
			},
			Spec: v1alpha2.MeshRootCertificateSpec{
				TrustDomain: "cluster.local",
				Intent:      intent,
				Provider: v1alpha2.ProviderSpec{
					CertManager: &v1alpha2.CertManagerProviderSpec{
						IssuerName:  name,
						IssuerKind:  "Issuer",
						IssuerGroup: "cert-manager.io",
					},
				},
			},
		}, metav1.CreateOptions{})
}

func createVaultMRC(name string, intent v1alpha2.MeshRootCertificateIntent) (*v1alpha2.MeshRootCertificate, error) {
	vaultPod, err := Td.GetPodsForLabel(Td.OsmNamespace, metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.AppLabel: "vault",
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(len(vaultPod)).Should(Equal(1))

	command := []string{"vault", "write", "pki/root/rotate/internal", "common_name=osm.root", fmt.Sprintf("issuer_name=%s", name)}
	stdout, stderr, err := Td.RunRemote(Td.OsmNamespace, vaultPod[0].Name, "vault", command)
	Td.T.Logf("Vault create new root output: %s, stderr:%s", stdout, stderr)
	Expect(err).NotTo(HaveOccurred())

	command = []string{"vault", "write", fmt.Sprintf("pki/roles/%s", name), "allow_any_name=true", "allow_subdomains=true", "max_ttl=87700h", "allowed_uri_sans=spiffe://*"}
	stdout, stderr, err = Td.RunRemote(Td.OsmNamespace, vaultPod[0].Name, "vault", command)
	Td.T.Logf("Vault create new role output: %s, stderr:%s", stdout, stderr)
	Expect(err).NotTo(HaveOccurred())

	return Td.ConfigClient.ConfigV1alpha2().MeshRootCertificates(Td.OsmNamespace).Create(
		context.Background(), &v1alpha2.MeshRootCertificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: Td.OsmNamespace,
			},
			Spec: v1alpha2.MeshRootCertificateSpec{
				TrustDomain: "cluster.local",
				Intent:      intent,
				Provider: v1alpha2.ProviderSpec{
					Vault: &v1alpha2.VaultProviderSpec{
						Host:     "vault." + Td.OsmNamespace + ".svc.cluster.local",
						Protocol: "http",
						Port:     8200,
						Role:     name,
						Token: v1alpha2.VaultTokenSpec{
							SecretKeyRef: v1alpha2.SecretKeyReferenceSpec{
								Name:      "osm-vault-token",
								Namespace: Td.OsmNamespace,
								Key:       "notused",
							}, // The test framework wires up the using default token right now so this isn't actually used
						},
					},
				},
			},
		}, metav1.CreateOptions{})
}
