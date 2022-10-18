package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	certman "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/tests/framework"
	. "github.com/openservicemesh/osm/tests/framework"
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

// basicCerRotationScenario rotates a new cert in through the follow pattern:
// | Step  | MRC1 old   | MRC2 new     | signer    | validator |
// | ----- | ---------- | ------------ | --------- | --------- |
// | 1     | active     |              | mcr1      | mcr1      |
// | 2     | active     | passive      | mcr1      | mcr2      |
// | 3     | active     | active       | mcr2/mrc1 | mcr1/mrc2 |
// | 4     | passive    | active       | mcr2      | mcr1      |
// | 5     | inactive   | active       | mcr2      | mcr2 	   |
// | 5     |            | active       | mcr2      | mcr2      |
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
	Expect(err.Error()).Should(ContainSubstring("cannot create MRC %s/%s with intent active. An MRC with active intent already exists in the control plane namespace", Td.OsmNamespace, activeNotAllowed))

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
	stdout, stderr, err := Td.RunLocal("kubectl", "rollout", "restart", "deployment", "osm-controller", "-n", Td.OsmNamespace)
	Td.T.Logf("stderr:\n%s\n", stderr)
	Td.T.Logf("stdout:\n%s\n", stdout)
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)
	Expect(Td.WaitForPodsRunningReady(Td.OsmNamespace, 60*time.Second, 1, nil)).To(Succeed())

	By("checking HTTP traffic for client -> server pod after restarting")
	verifyCertRotation(clientPod, serverPod, newCertName, newCertName)
	verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)

	// todo maybe check that cert modules are different?
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
								Key:       "notused",
							}, // The test framework wires up the using default token right now so this isn't actually used
						},
					},
				},
			},
		}, metav1.CreateOptions{})
}

/*func verifiyUpdatedPodCert(pod *v1.Pod) {
	By("Verifying pod has updated certificates")

	// It can take a moment for envoy to load the certs
	Eventually(func() (string, error) {
		args := []string{"proxy", "get", "certs", pod.Name, fmt.Sprintf("-n=%s", pod.Namespace)}
		stdout, _, err := Td.RunLocal(filepath.FromSlash("../../bin/osm"), args...)
		Td.T.Logf("stdout:\n%s", stdout)
		return stdout.String(), err
	}, 10*time.Second).Should(ContainSubstring(fmt.Sprintf("\"uri\": \"spiffe://cluster.local/%s/%s", pod.Spec.ServiceAccountName, pod.Namespace)))
}*/

func verifySuccessfulPodConnection(srcPod, dstPod *v1.Pod, serverSvc *v1.Service) {
	By("Waiting for repeated request success")
	cond := Td.WaitForRepeatedSuccess(func() bool {
		result :=
			Td.FortioHTTPLoadTest(FortioHTTPLoadTestDef{
				HTTPRequestDef: HTTPRequestDef{
					SourceNs:        srcPod.Namespace,
					SourcePod:       srcPod.Name,
					SourceContainer: srcPod.Name,

					Destination: fmt.Sprintf("%s.%s:%d", serverSvc.Name, dstPod.Namespace, fortioHTTPPort),
				},
			})

		if result.Err != nil || result.HasFailedHTTPRequests() {
			Td.T.Logf("> REST req has failed requests: %v", result.Err)
			return false
		}
		Td.T.Logf("> REST req succeeded. Status codes: %v", result.AllReturnCodes())
		return true
	}, 2 /*runs the load test this many times successfully*/, 90*time.Second /*timeout*/)
	Expect(cond).To(BeTrue())
}

// verifyCertRotation ensure the certificates have been rotated properly across all components:
// 1. Verify service certs were updated (todo)
// 2. Verify webhooks were updated (todo)
// 3. Verify bootstrap certs updated (todo)
// 4. verify xds cert updated (todo)
func verifyCertRotation(clientPodDef, serverPodDef *v1.Pod, signingCertName, validatingCertName string) {
	By("checking bootstrap secrets are updated after creating MRC with passive intent")
	podSelector := constants.EnvoyUniqueIDLabelName
	srvPod, err := Td.Client.CoreV1().Pods(serverPodDef.Namespace).Get(context.Background(), serverPodDef.Name, metav1.GetOptions{})
	Expect(err).To(BeNil())

	clientPod, err := Td.Client.CoreV1().Pods(clientPodDef.Namespace).Get(context.Background(), clientPodDef.Name, metav1.GetOptions{})
	Expect(err).To(BeNil())

	srvPodUUID := srvPod.GetLabels()[podSelector]
	clientPodUUID := clientPod.GetLabels()[podSelector]

	srvSecretName := fmt.Sprintf("envoy-bootstrap-config-%s", srvPodUUID)
	clientSecretName := fmt.Sprintf("envoy-bootstrap-config-%s", clientPodUUID)

	// TODO(jaellio): add a time.wait instead of waiting so long in call
	err = Td.WaitForBootstrapSecretUpdate(serverPodDef.Namespace, srvSecretName, signingCertName, validatingCertName, time.Second*30)
	Expect(err).NotTo(HaveOccurred())
	err = Td.WaitForBootstrapSecretUpdate(clientPodDef.Namespace, clientSecretName, signingCertName, validatingCertName, time.Second*30)
	Expect(err).NotTo(HaveOccurred())
}

// func getCertificateModulus(pod *v1.Pod) string {
// 	// It can take a moment for envoy to load the certs
// 	Eventually(func() (string, error) {
// 		args := []string{"proxy", "get", "config_dump", pod.Name, fmt.Sprintf("-n=%s", pod.Namespace)}
// 		stdout, _, err := Td.RunOsmCli(args...)

// 		return stdout.String(), err
// 	}, 10*time.Second).Should(ContainSubstring(fmt.Sprintf("\"uri\": \"spiffe://cluster.local/%s/%s", pod.Spec.ServiceAccountName, pod.Namespace)))
// }

func deployTestWorkload() (*v1.Pod, *v1.Pod, *v1.Service) {
	var (
		clientNamespace = framework.RandomNameWithPrefix("client")
		serverNamespace = framework.RandomNameWithPrefix("server")
		ns              = []string{clientNamespace, serverNamespace}
	)

	By("Deploying client -> server workload")
	// Create namespaces
	for _, n := range ns {
		Expect(Td.CreateNs(n, nil)).To(Succeed())
		Expect(Td.AddNsToMesh(true, n)).To(Succeed())
	}

	// Get simple pod definitions for the HTTP server
	serverSvcAccDef, serverPodDef, serverSvcDef, err := Td.SimplePodApp(
		SimplePodAppDef{
			PodName:   framework.RandomNameWithPrefix("pod"),
			Namespace: serverNamespace,
			Image:     fortioImageName,
			Ports:     []int{fortioHTTPPort},
			OS:        Td.ClusterOS,
		})
	Expect(err).NotTo(HaveOccurred())

	_, err = Td.CreateServiceAccount(serverNamespace, &serverSvcAccDef)
	Expect(err).NotTo(HaveOccurred())
	serverPod, err := Td.CreatePod(serverNamespace, serverPodDef)
	Expect(err).NotTo(HaveOccurred())
	serverSvc, err := Td.CreateService(serverNamespace, serverSvcDef)
	Expect(err).NotTo(HaveOccurred())

	// Expect it to be up and running in it's receiver namespace
	Expect(Td.WaitForPodsRunningReady(serverNamespace, 60*time.Second, 1, nil)).To(Succeed())

	// Get simple Pod definitions for the client
	podName := framework.RandomNameWithPrefix("pod")
	clientSvcAccDef, clientPodDef, clientSvcDef, err := Td.SimplePodApp(SimplePodAppDef{
		PodName:       podName,
		Namespace:     clientNamespace,
		ContainerName: podName,
		Image:         fortioImageName,
		Ports:         []int{fortioHTTPPort},
		OS:            Td.ClusterOS,
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = Td.CreateServiceAccount(clientNamespace, &clientSvcAccDef)
	Expect(err).NotTo(HaveOccurred())
	clientPod, err := Td.CreatePod(clientNamespace, clientPodDef)
	Expect(err).NotTo(HaveOccurred())
	_, err = Td.CreateService(clientNamespace, clientSvcDef)
	Expect(err).NotTo(HaveOccurred())

	// Expect it to be up and running in it's receiver namespace
	Expect(Td.WaitForPodsRunningReady(clientNamespace, 60*time.Second, 1, nil)).To(Succeed())

	// Deploy allow rule client->server
	httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
		SimpleAllowPolicy{
			RouteGroupName:    "routes",
			TrafficTargetName: "target",

			SourceNamespace:      clientNamespace,
			SourceSVCAccountName: clientSvcAccDef.Name,

			DestinationNamespace:      serverNamespace,
			DestinationSvcAccountName: serverSvcAccDef.Name,
		})

	// Configs have to be put into a monitored NS, and osm-system can't be by cli
	_, err = Td.CreateHTTPRouteGroup(serverNamespace, httpRG)
	Expect(err).NotTo(HaveOccurred())
	_, err = Td.CreateTrafficTarget(serverNamespace, trafficTarget)
	Expect(err).NotTo(HaveOccurred())

	return clientPod, serverPod, serverSvc
}
