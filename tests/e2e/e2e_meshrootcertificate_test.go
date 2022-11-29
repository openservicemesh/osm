package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	certman "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("MeshRootCertificate",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 11,
	},
	func() {
		Context("with Tresor", func() {
			It("rotates certificates", func() {
				basicCertRotationScenario()
			})

			It("handles enabling MRC after install", func() {
				enablingMRCAfterInstallScenario()
			})

			It("rotates trust domains", func() {
				trustDomainRotation()
			})
		})
	})

var _ = OSMDescribe("MeshRootCertificate",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 13,
	},
	func() {
		Context("with CertManager", func() {
			It("rotates certificates", func() {
				basicCertRotationScenario(WithCertManagerEnabled())
			})

			It("handles enabling MRC after install", func() {
				enablingMRCAfterInstallScenario(WithCertManagerEnabled())
			})
		})
	})

var _ = OSMDescribe("MeshRootCertificate",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 14,
	},
	func() {
		Context("with Vault", func() {
			It("rotates certificates", func() {
				basicCertRotationScenario(WithVault())
			})

			It("handles enabling MRC after install", func() {
				enablingMRCAfterInstallScenario(WithVault(), WithVaultTokenSecretRef())
			})
		})
	})

var _ = OSMDescribe("MeshRootCertificate",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 15,
	},
	func() {
		Context("can switch providers", func() {
			It("during rotation", func() {
				providerChangeRotation()
			})
		})
	})

// providerChangeRotation rotates the certificate using the osm cli rotation command
// and switches providers tresor -> cert-manager -> vault -> tresor
func providerChangeRotation(installOptions ...InstallOsmOpt) {
	By("installing with MRC enabled")
	installOptions = append(installOptions, WithMeshRootCertificateEnabled())
	installOpts := Td.GetOSMInstallOpts(installOptions...)
	Expect(Td.InstallOSM(installOpts)).To(Succeed())

	// install these so we can use them later
	err := Td.InstallCertManager()
	Expect(err).NotTo(HaveOccurred())
	err = Td.InstallVault(installOpts)
	Expect(err).NotTo(HaveOccurred())

	By("checking HTTP traffic for client -> server pod after initial MRC creation")
	clientPod, serverPod, serverSvc := deployTestWorkload()
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	By("creating new MRC with CertManager configuration")
	certManagerMRC := "cert-manager"
	_, err = createMeshRootCertificate(certManagerMRC, v1alpha2.InactiveIntent, CertManager)
	Expect(err).NotTo(HaveOccurred())

	By("rotating the certificate to CertManager")
	args := []string{"alpha", "certificate", "rotate", "-y", "-d", "-w", "35s", "-c", certManagerMRC}
	stdout, _, err := Td.RunOsmCli(args...)
	Td.T.Logf("stdout:\n%s", stdout)
	Expect(err).NotTo(HaveOccurred())

	By("checking HTTP traffic for client -> server pod after CertManager cert is rotated in")
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	By("creating new MRC with Vault configuration")
	vaultMRC := "vault"
	_, err = createMeshRootCertificate(vaultMRC, v1alpha2.InactiveIntent, Vault)
	Expect(err).NotTo(HaveOccurred())

	By("rotating the certificate to Vault")
	args = []string{"alpha", "certificate", "rotate", "-y", "-d", "-w", "35s", "-c", vaultMRC}
	stdout, _, err = Td.RunOsmCli(args...)
	Td.T.Logf("stdout:\n%s", stdout)
	Expect(err).NotTo(HaveOccurred())

	By("checking HTTP traffic for client -> server pod after Vault cert is rotated in")
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	By("creating new MRC with Tresor configuration")
	tresorMRC := "tresor"
	_, err = createMeshRootCertificate(tresorMRC, v1alpha2.InactiveIntent, TresorCertManager)
	Expect(err).NotTo(HaveOccurred())

	By("rotating the certificate to Tresor")
	args = []string{"alpha", "certificate", "rotate", "-y", "-d", "-w", "35s", "-c", tresorMRC}
	stdout, _, err = Td.RunOsmCli(args...)
	Td.T.Logf("stdout:\n%s", stdout)
	Expect(err).NotTo(HaveOccurred())

	By("checking HTTP traffic for client -> server pod after CertManager cert is rotated in")
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)
}

// trustDomainRotation rotates the certificate using the osm cli rotation command
func trustDomainRotation(installOptions ...InstallOsmOpt) {
	By("installing with MRC enabled")
	installOptions = append(installOptions, WithMeshRootCertificateEnabled())
	installOpts := Td.GetOSMInstallOpts(installOptions...)
	Expect(Td.InstallOSM(installOpts)).To(Succeed())

	By("checking HTTP traffic for client -> server pod after initial MRC creation")
	clientPod, serverPod, serverSvc := deployTestWorkload()
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	// There are 4 steps to the rotation and we set the rotation propagation wait time to 20s
	durationOfLoadTest := "2m30s"
	resultChn := make(chan FortioLoadResult)
	go func() {
		result := Td.FortioHTTPLoadTest(FortioHTTPLoadTestDef{
			HTTPRequestDef: HTTPRequestDef{
				SourceNs:        clientPod.Namespace,
				SourcePod:       clientPod.Name,
				SourceContainer: clientPod.Name,

				Destination: fmt.Sprintf("%s.%s:%d", serverSvc.Name, serverPod.Namespace, fortioHTTPPort),
			}, FortioLoadTestSpec: FortioLoadTestSpec{Duration: durationOfLoadTest}})
		resultChn <- result
	}()

	By("rotating the certificate with a new trustdomain")
	args := []string{"alpha", "certificate", "rotate", "-y", "-d", "-t", "cluster.new", "-w", "35s"}
	stdout, _, err := Td.RunOsmCli(args...)
	Td.T.Logf("stdout:\n%s", stdout)
	Expect(err).NotTo(HaveOccurred())

	By("By checking results of load test run during rotation")
	result := <-resultChn
	Expect(result.Err).NotTo(HaveOccurred())
	if result.ReturnCodes["200"].Percentage < .99 {
		Fail(fmt.Sprintf("To many requested failed. Success rate: %f", result.ReturnCodes["200"].Percentage*100))
	}
	Td.T.Logf("> REST req succeeded. Status codes: %v", result.AllReturnCodes())

	By("By verifying the default secret is not present")
	_, err = Td.Client.CoreV1().Secrets(Td.OsmNamespace).Get(context.Background(), OsmCABundleName, metav1.GetOptions{})
	Expect(err).Should(HaveOccurred())

	By("By verifying the new trust domain")
	args = []string{"proxy", "get", "certs", clientPod.Name, fmt.Sprintf("-n=%s", clientPod.Namespace)}
	stdout, _, err = Td.RunOsmCli(args...)
	Td.T.Logf("stdout:\n%s", stdout)
	Expect(err).NotTo(HaveOccurred())
	Expect(stdout.String()).Should(ContainSubstring(fmt.Sprintf("%s.%s.cluster.new", clientPod.Spec.ServiceAccountName, clientPod.Namespace)))
}

// basicCerRotationScenario rotates a new cert in through the follow pattern:
// | Step  | MRC1 old   | MRC2 new     | signer    | validator |
// | ----- | ---------- | ------------ | --------- | --------- |
// | 1     | active     |              | mrc1      | mrc1      |
// | 2     | active     | passive      | mrc1      | mrc2      |
// | 3     | active     | active       | mrc2/mrc1 | mrc1/mrc2 |
// | 4     | passive    | active       | mrc2      | mrc1      |
// | 5     | inactive   | active       | mrc2      | mrc2 	   |
// | 5     |            | active       | mrc2      | mrc2      |
func basicCertRotationScenario(installOptions ...InstallOsmOpt) {
	By("installing with MRC enabled")
	installOptions = append(installOptions, WithMeshRootCertificateEnabled())
	installOpts := Td.GetOSMInstallOpts(installOptions...)
	Expect(Td.InstallOSM(installOpts)).To(Succeed())

	By("checking the certificate exists")
	if installOpts.CertManager != Vault {
		// no secrets are created in Vault case
		err := Td.WaitForCABundleSecret(Td.OsmNamespace, OsmCABundleName, time.Second*5)
		Expect(err).NotTo(HaveOccurred())
	}

	By("checking HTTP traffic for client -> server pod after initial MRC creation")
	clientPod, serverPod, serverSvc := deployTestWorkload()
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	By("checking that another cert with active intent cannot be created")
	time.Sleep(time.Second * 10)
	activeNotAllowed := "not-allowed"
	_, err := createMeshRootCertificate(activeNotAllowed, v1alpha2.ActiveIntent, installOpts.CertManager)
	Expect(err).Should(HaveOccurred())
	Expect(err.Error()).Should(ContainSubstring("cannot create MRC %s/%s with active intent. An MRC with this intent already exists in the control plane namespace.", Td.OsmNamespace, activeNotAllowed))

	By("creating a second certificate with passive intent")
	newCertName := "osm-mrc-2"
	_, err = createMeshRootCertificate(newCertName, v1alpha2.PassiveIntent, installOpts.CertManager)
	Expect(err).NotTo(HaveOccurred())

	By("ensuring the new CA secret exists")
	if installOpts.CertManager != Vault {
		// no secrets are created in Vault case
		err = Td.WaitForCABundleSecret(Td.OsmNamespace, newCertName, time.Second*90)
		Expect(err).NotTo(HaveOccurred())
	}

	By("checking HTTP traffic for client -> server pod after new cert is rotated in")
	verifyCertRotation(clientPod, serverPod, constants.DefaultMeshRootCertificateName, newCertName)
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	By("moving new cert from Passive to Active")
	updateCertificate(newCertName, v1alpha2.ActiveIntent)

	By("checking HTTP traffic for client -> server pod after new cert is rotated in for validation")
	// At this stage when they are both same intent it isn't deterministic which is validating and signing
	// skip the verification for now. We check it again later.
	// verifyCertRotation(clientPod, serverPod, newCertName, constants.DefaultMeshRootCertificateName)
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	By("moving original cert from Active to Passive")
	updateCertificate(constants.DefaultMeshRootCertificateName, v1alpha2.PassiveIntent)

	By("checking HTTP traffic for client -> server pod after new cert is rotated in for signing")
	verifyCertRotation(clientPod, serverPod, newCertName, constants.DefaultMeshRootCertificateName)
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	By("moving original cert from passive to inactive")
	updateCertificate(constants.DefaultMeshRootCertificateName, v1alpha2.InactiveIntent)

	By("checking HTTP traffic for client -> server pod after removing original cert")
	verifyCertRotation(clientPod, serverPod, newCertName, newCertName)
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	// Success! the new certificate is in use; lets go nuclear and make sure
	By("deleting original certificate")
	err = Td.ConfigClient.ConfigV1alpha2().MeshRootCertificates(Td.OsmNamespace).Delete(context.Background(), constants.DefaultMeshRootCertificateName, metav1.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())
	if installOpts.CertManager != Vault {
		// no secrets are created in Vault case
		err = Td.Client.CoreV1().Secrets(Td.OsmNamespace).Delete(context.Background(), OsmCABundleName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	}

	By("checking HTTP traffic for client -> server pod after deleting original cert")
	verifyCertRotation(clientPod, serverPod, newCertName, newCertName)
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	By("restarting osm controller")
	// If anything got stuck in above rotation this will ensure things are indeed working
	err = Td.RolloutRestartOSMControlPlaneComponent(constants.OSMControllerName)
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)
	Expect(Td.WaitForPodsRunningReady(Td.OsmNamespace, 60*time.Second, 1, nil)).To(Succeed())

	By("checking HTTP traffic for client -> server pod after restarting")
	verifyCertRotation(clientPod, serverPod, newCertName, newCertName)
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	// todo maybe check that cert modules are different?
}

func enablingMRCAfterInstallScenario(installOptions ...InstallOsmOpt) {
	By("installing with MRC disabled")
	installOpts := Td.GetOSMInstallOpts(installOptions...)
	Expect(Td.InstallOSM(installOpts)).To(Succeed())

	By("checking the certificate exists")
	if installOpts.CertManager != Vault {
		// no secrets are created in Vault case
		err := Td.WaitForCABundleSecret(Td.OsmNamespace, OsmCABundleName, time.Second*5)
		Expect(err).NotTo(HaveOccurred())
	}

	By("checking that an MRC was not created")
	time.Sleep(time.Second * 10)
	_, err := Td.ConfigClient.ConfigV1alpha2().MeshRootCertificates(Td.OsmNamespace).Get(
		context.Background(), constants.DefaultMeshRootCertificateName, metav1.GetOptions{})
	Expect(err).Should(HaveOccurred())
	Expect(apierrors.IsNotFound(err)).To(BeTrue())

	By("checking HTTP traffic for client -> server pod")
	clientPod, serverPod, serverSvc := deployTestWorkload()
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	By("enabling EnableMeshRootCertificate feature flag by setting the flag in the MeshConfig")
	meshConfig, _ := Td.GetMeshConfig(Td.OsmNamespace)
	meshConfig.Spec.FeatureFlags.EnableMeshRootCertificate = true
	updatedMeshConfig, err := Td.UpdateOSMConfig(meshConfig)
	Expect(err).NotTo(HaveOccurred())
	Expect(updatedMeshConfig.Spec.FeatureFlags.EnableMeshRootCertificate).To(BeTrue())

	By("restarting the osm-bootstrap")
	err = Td.RolloutRestartOSMControlPlaneComponent(constants.OSMBootstrapName)
	Expect(err).NotTo(HaveOccurred())

	By("checking that an active MRC was created")
	Eventually(func() error {
		_, err := Td.ConfigClient.ConfigV1alpha2().MeshRootCertificates(Td.OsmNamespace).Get(
			context.Background(), constants.DefaultMeshRootCertificateName, metav1.GetOptions{})
		return err
	}, 10*time.Second).Should(BeNil())

	By("restarting the osm-injector and osm-controller")
	err = Td.RolloutRestartOSMControlPlaneComponent(constants.OSMControllerName)
	Expect(err).NotTo(HaveOccurred())
	err = Td.RolloutRestartOSMControlPlaneComponent(constants.OSMInjectorName)
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)

	By("checking HTTP traffic for client -> server pod")
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)
}

func createMeshRootCertificate(name string, intent v1alpha2.MeshRootCertificateIntent, certificateManagerType string) (*v1alpha2.MeshRootCertificate, error) {
	switch certificateManagerType {
	case DefaultCertManager:
		return createTresorMRC(name, intent)
	case CertManager:
		return createCertManagerMRC(name, intent)
	case Vault:
		return createVaultMRC(name, intent)
	default:
		Fail("should not be able to create MRC of unknown type")
		return nil, fmt.Errorf("should not be able to create MRC of unknown type")
	}
}

func createTresorMRC(name string, intent v1alpha2.MeshRootCertificateIntent) (*v1alpha2.MeshRootCertificate, error) {
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

func updateCertificate(name string, intent v1alpha2.MeshRootCertificateIntent) {
	mrc, err := Td.ConfigClient.ConfigV1alpha2().MeshRootCertificates(Td.OsmNamespace).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	mrc.Spec.Intent = intent

	_, err = Td.ConfigClient.ConfigV1alpha2().MeshRootCertificates(Td.OsmNamespace).Update(context.Background(), mrc, metav1.UpdateOptions{})
	Expect(err).NotTo(HaveOccurred())
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
								Key:       "vault_token",
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
}

// verifyCertRotation ensure the certificates have been rotated properly across all components:
// 1. Verify service certs were updated (todo)
// 2. Verify webhooks were updated (todo)
// 3. Verify bootstrap certs updated (todo)
// 4. verify xds cert updated (todo)
func verifyCertRotation(clientPodDef, serverPodDef *v1.Pod, signingCertName, validatingCertName string) {
	By("checking bootstrap secrets are updated")
	podSelector := constants.EnvoyUniqueIDLabelName
	srvPod, err := Td.Client.CoreV1().Pods(serverPodDef.Namespace).Get(context.Background(), serverPodDef.Name, metav1.GetOptions{})
	Expect(err).To(BeNil())

	clientPod, err := Td.Client.CoreV1().Pods(clientPodDef.Namespace).Get(context.Background(), clientPodDef.Name, metav1.GetOptions{})
	Expect(err).To(BeNil())

	srvPodUUID := srvPod.GetLabels()[podSelector]
	clientPodUUID := clientPod.GetLabels()[podSelector]

	srvSecretName := fmt.Sprintf("envoy-bootstrap-config-%s", srvPodUUID)
	clientSecretName := fmt.Sprintf("envoy-bootstrap-config-%s", clientPodUUID)

	err = Td.WaitForBootstrapSecretUpdate(serverPodDef.Namespace, srvSecretName, signingCertName, validatingCertName, time.Second*30)
	Expect(err).NotTo(HaveOccurred())
	err = Td.WaitForBootstrapSecretUpdate(clientPodDef.Namespace, clientSecretName, signingCertName, validatingCertName, time.Second*30)
	Expect(err).NotTo(HaveOccurred())
}
